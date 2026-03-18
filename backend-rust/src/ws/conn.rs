use std::collections::HashMap;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;

use axum::extract::ws::{Message, WebSocket};
use futures_util::{SinkExt, StreamExt};
use serde::Serialize;
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;
use tracing::{debug, warn};

use super::protocol::{AckMessage, ClientMessage, ServerMessage};

static CONN_ID_COUNTER: AtomicU64 = AtomicU64::new(0);

const WRITE_CHANNEL_SIZE: usize = 64;

/// A WebSocket connection.
pub struct Conn {
    pub id: String,
    user_id: std::sync::atomic::AtomicI32,
    tx: mpsc::Sender<Message>,
    subscriptions: std::sync::Mutex<HashMap<&'static str, CancellationToken>>,
}

impl Conn {
    pub fn user_id(&self) -> i32 {
        self.user_id.load(Ordering::Relaxed)
    }

    pub fn set_user(&self, id: i32) {
        self.user_id.store(id, Ordering::Relaxed);
    }

    pub fn is_authenticated(&self) -> bool {
        self.user_id() != 0
    }

    /// Send a raw WebSocket message.
    pub async fn send(&self, msg: Message) -> bool {
        self.tx.send(msg).await.is_ok()
    }

    /// Send a JSON ack response.
    pub async fn send_ack<T: Serialize>(&self, id: i64, data: T) {
        let ack = AckMessage { id, data };
        match serde_json::to_string(&ack) {
            Ok(json) => {
                let _ = self.send(Message::Text(json.into())).await;
            }
            Err(e) => warn!(conn = %self.id, "failed to serialize ack: {e}"),
        }
    }

    /// Send a push event. Returns true if sent successfully.
    pub async fn send_event<T: Serialize>(&self, event: &str, data: T) -> bool {
        let msg = ServerMessage {
            event: event.to_string(),
            data,
        };
        match serde_json::to_string(&msg) {
            Ok(json) => self.send(Message::Text(json.into())).await,
            Err(e) => {
                warn!(conn = %self.id, "failed to serialize event: {e}");
                false
            }
        }
    }

    /// Cancel an existing subscription by key, then store a new token.
    pub fn set_subscription(&self, key: &'static str, token: CancellationToken) {
        let mut subs = self.subscriptions.lock().unwrap();
        if let Some(old) = subs.insert(key, token) {
            old.cancel();
        }
    }

    /// Cancel and remove a subscription by key.
    pub fn cancel_subscription(&self, key: &'static str) {
        let mut subs = self.subscriptions.lock().unwrap();
        if let Some(token) = subs.remove(key) {
            token.cancel();
        }
    }

    /// Cancel all subscriptions (called on disconnect).
    pub fn cancel_all_subscriptions(&self) {
        let mut subs = self.subscriptions.lock().unwrap();
        for (_, token) in subs.drain() {
            token.cancel();
        }
    }

    /// Close the connection by sending a Close frame.
    pub fn close(&self) {
        let tx = self.tx.clone();
        tokio::spawn(async move {
            let _ = tx.send(Message::Close(None)).await;
        });
    }
}

/// Handler function type for connect events.
pub type ConnectFn = Box<dyn Fn(std::sync::Arc<Conn>) + Send + Sync + 'static>;

/// Handler function type for binary frames.
/// data is [opcode:1][payload:N] (session header already stripped).
pub type BinaryFn =
    Box<dyn Fn(std::sync::Arc<Conn>, u16, &[u8]) + Send + Sync + 'static>;

/// Splits a WebSocket into read/write halves and runs the connection.
/// Returns the connection Arc for registration.
pub fn spawn_conn(
    socket: WebSocket,
    server: std::sync::Arc<super::WsServer>,
    broadcast_rx: tokio::sync::broadcast::Receiver<Arc<str>>,
) -> std::sync::Arc<Conn> {
    let id_num = CONN_ID_COUNTER.fetch_add(1, Ordering::Relaxed) + 1;
    let conn_id = format!("c{id_num}");

    let (tx, rx) = mpsc::channel(WRITE_CHANNEL_SIZE);

    let conn = std::sync::Arc::new(Conn {
        id: conn_id,
        user_id: std::sync::atomic::AtomicI32::new(0),
        tx,
        subscriptions: std::sync::Mutex::new(HashMap::new()),
    });

    let (ws_write, ws_read) = socket.split();

    // Write pump: select over direct mpsc + broadcast channel
    let conn_w = conn.clone();
    tokio::spawn(write_pump(conn_w, rx, broadcast_rx, ws_write));

    // Read pump
    let conn_r = conn.clone();
    tokio::spawn(read_pump(conn_r, ws_read, server));

    conn
}

async fn write_pump(
    conn: std::sync::Arc<Conn>,
    mut rx: mpsc::Receiver<Message>,
    mut broadcast_rx: tokio::sync::broadcast::Receiver<Arc<str>>,
    mut ws_write: futures_util::stream::SplitSink<WebSocket, Message>,
) {
    loop {
        tokio::select! {
            msg = rx.recv() => {
                match msg {
                    Some(msg) => {
                        let is_close = matches!(&msg, Message::Close(_));
                        if ws_write.send(msg).await.is_err() {
                            debug!(conn = %conn.id, "write error, closing");
                            break;
                        }
                        if is_close {
                            debug!(conn = %conn.id, "close frame sent, shutting down write pump");
                            break;
                        }
                    }
                    None => break,
                }
            }
            result = broadcast_rx.recv() => {
                match result {
                    Ok(payload) => {
                        // Only forward broadcasts to authenticated connections
                        if conn.is_authenticated()
                            && ws_write.send(Message::Text((*payload).into())).await.is_err()
                        {
                            break;
                        }
                    }
                    Err(tokio::sync::broadcast::error::RecvError::Lagged(n)) => {
                        warn!(conn = %conn.id, lagged = n, "broadcast receiver lagged");
                    }
                    Err(tokio::sync::broadcast::error::RecvError::Closed) => break,
                }
            }
        }
    }
    let _ = ws_write.close().await;
}

async fn read_pump(
    conn: std::sync::Arc<Conn>,
    mut ws_read: futures_util::stream::SplitStream<WebSocket>,
    server: std::sync::Arc<super::WsServer>,
) {
    // Use the global dispatch semaphore for backpressure
    let sem = server.dispatch_semaphore.clone();

    while let Some(Ok(msg)) = ws_read.next().await {
        match msg {
            Message::Text(text) => {
                let parsed: Result<ClientMessage, _> = serde_json::from_str(&text);
                match parsed {
                    Ok(client_msg) => {
                        let conn = conn.clone();
                        let server = server.clone();
                        let sem = sem.clone();
                        tokio::spawn(async move {
                            let _permit = sem.acquire().await;
                            server.dispatch(conn, client_msg).await;
                        });
                    }
                    Err(e) => {
                        warn!(conn = %conn.id, "malformed message: {e}");
                    }
                }
            }
            Message::Binary(data) => {
                if data.len() < 3 {
                    continue;
                }
                let session_id = u16::from_be_bytes([data[0], data[1]]);
                let payload = &data[2..];
                if let Some(ref handler) = server.binary_handler {
                    handler(conn.clone(), session_id, payload);
                }
            }
            Message::Close(_) => break,
            _ => {}
        }
    }

    // Connection closed — remove from registry
    server.remove(&conn);
}
