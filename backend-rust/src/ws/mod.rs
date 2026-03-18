pub mod conn;
pub mod protocol;

use std::collections::HashMap;
use std::sync::Arc;

use std::sync::RwLock;

use axum::extract::ws::WebSocket;
use tracing::{debug, warn};

use conn::{BinaryFn, Conn, ConnectFn};
use protocol::ErrorResponse;

/// Handler function type for disconnect events.
pub type DisconnectFn = Box<dyn Fn(&Conn) + Send + Sync + 'static>;

type AsyncHandlerFn = Box<
    dyn Fn(Arc<Conn>, protocol::ClientMessage) -> std::pin::Pin<Box<dyn std::future::Future<Output = ()> + Send>>
        + Send
        + Sync
        + 'static,
>;

/// Central WebSocket server: connection registry + handler dispatch.
pub struct WsServer {
    conns: RwLock<HashMap<String, Arc<Conn>>>,
    handlers: HashMap<String, AsyncHandlerFn>,
    connect_handler: Option<ConnectFn>,
    disconnect_handler: Option<DisconnectFn>,
    pub(crate) binary_handler: Option<BinaryFn>,
    /// Global dispatch semaphore shared across all connections for backpressure.
    pub(crate) dispatch_semaphore: Arc<tokio::sync::Semaphore>,
    /// Broadcast channel for push events to all authenticated connections.
    broadcaster: crate::broadcast::Broadcaster,
}

impl WsServer {
    pub fn new(broadcaster: crate::broadcast::Broadcaster) -> Self {
        Self {
            conns: RwLock::new(HashMap::new()),
            handlers: HashMap::new(),
            connect_handler: None,
            disconnect_handler: None,
            binary_handler: None,
            dispatch_semaphore: Arc::new(tokio::sync::Semaphore::new(64)),
            broadcaster,
        }
    }

    /// Register a handler for a named event.
    pub fn handle<F, Fut>(&mut self, event: &str, handler: F)
    where
        F: Fn(Arc<Conn>, protocol::ClientMessage) -> Fut + Send + Sync + 'static,
        Fut: std::future::Future<Output = ()> + Send + 'static,
    {
        let handler = Arc::new(handler);
        self.handlers.insert(
            event.to_string(),
            Box::new(move |conn, msg| {
                let handler = handler.clone();
                Box::pin(async move {
                    handler(conn, msg).await;
                })
            }),
        );
    }

    /// Register a handler called when a new connection is established.
    pub fn handle_connect<F>(&mut self, handler: F)
    where
        F: Fn(Arc<Conn>) + Send + Sync + 'static,
    {
        self.connect_handler = Some(Box::new(handler));
    }

    /// Register a handler called when a connection is closed.
    pub fn handle_disconnect<F>(&mut self, handler: F)
    where
        F: Fn(&Conn) + Send + Sync + 'static,
    {
        self.disconnect_handler = Some(Box::new(handler));
    }

    /// Accept a new WebSocket connection.
    pub fn accept(self: &Arc<Self>, socket: WebSocket) {
        let broadcast_rx = self.broadcaster.subscribe();
        let conn = conn::spawn_conn(socket, self.clone(), broadcast_rx);

        // Register connection
        {
            let mut conns = self.conns.write().unwrap();
            conns.insert(conn.id.clone(), conn.clone());
        }

        // Fire connect handler
        if let Some(ref handler) = self.connect_handler {
            handler(conn);
        }
    }

    /// Remove a connection from the registry.
    pub fn remove(&self, conn: &Conn) {
        debug!(conn = %conn.id, "connection removed");

        // Fire disconnect handler before removing
        if let Some(ref handler) = self.disconnect_handler {
            handler(conn);
        }

        self.conns.write().unwrap().remove(&conn.id);
    }

    /// Dispatch a message to the appropriate handler.
    pub async fn dispatch(&self, conn: Arc<Conn>, msg: protocol::ClientMessage) {
        if let Some(handler) = self.handlers.get(&msg.event) {
            handler(conn, msg).await;
        } else {
            warn!(event = %msg.event, "unknown event");
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::new(format!("unknown event: {}", msg.event)))
                    .await;
            }
        }
    }

    /// Disconnect all connections except the given one.
    fn disconnect_others(&self, keep_conn_id: &str) {
        let conns = self.conns.read().unwrap();
        for conn in conns.values() {
            if conn.id != keep_conn_id {
                conn.close();
            }
        }
    }

    /// Spawn a task that processes control messages (e.g., disconnect others).
    pub fn spawn_control_loop(
        self: &Arc<Self>,
        mut rx: tokio::sync::mpsc::Receiver<crate::broadcast::WsControlMsg>,
    ) {
        let server = self.clone();
        tokio::spawn(async move {
            while let Some(msg) = rx.recv().await {
                match msg {
                    crate::broadcast::WsControlMsg::DisconnectOthers { keep_conn_id, done } => {
                        server.disconnect_others(&keep_conn_id);
                        let _ = done.send(());
                    }
                }
            }
        });
    }
}
