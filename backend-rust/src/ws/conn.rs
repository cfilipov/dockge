use std::collections::HashMap;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;

use axum::extract::ws::{Message, WebSocket};
use futures_util::{SinkExt, StreamExt};
use serde::Serialize;
use tokio::sync::{broadcast, mpsc};
use tokio_util::sync::CancellationToken;
use tracing::{debug, warn};

use super::protocol::{AckMessage, ClientMessage, ServerMessage};

static CONN_ID_COUNTER: AtomicU64 = AtomicU64::new(0);

/// A WebSocket connection.
pub struct Conn {
    pub id: String,
    user_id: std::sync::atomic::AtomicI32,
    tx: mpsc::UnboundedSender<Message>,
    /// Keyed by subscription type ("stats", "top"). Value is (resource_id, token).
    /// The resource_id (e.g. container name) prevents a late-arriving unsubscribe
    /// from cancelling a newer subscription for a different resource.
    subscriptions: std::sync::Mutex<HashMap<&'static str, (String, CancellationToken)>>,
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

    /// Send a raw WebSocket message (non-blocking).
    pub fn send(&self, msg: Message) -> bool {
        self.tx.send(msg).is_ok()
    }

    /// Alias for `send` — with unbounded channel, all sends are non-blocking.
    pub fn send_nowait(&self, msg: Message) -> bool {
        self.send(msg)
    }

    /// Send a JSON ack response.
    pub fn send_ack<T: Serialize>(&self, id: i64, data: T) {
        let ack = AckMessage { id, data };
        match serde_json::to_string(&ack) {
            Ok(json) => {
                let _ = self.send(Message::Text(json.into()));
            }
            Err(e) => warn!(conn = %self.id, "failed to serialize ack: {e}"),
        }
    }

    /// Send a push event. Returns true if sent successfully.
    pub fn send_event<T: Serialize>(&self, event: &str, data: T) -> bool {
        let msg = ServerMessage {
            event: event.to_string(),
            data,
        };
        match serde_json::to_string(&msg) {
            Ok(json) => self.send(Message::Text(json.into())),
            Err(e) => {
                warn!(conn = %self.id, "failed to serialize event: {e}");
                false
            }
        }
    }

    /// Synchronous variant of `send_event` — identical to `send_event` now
    /// that all sends are non-blocking. Kept for API compatibility.
    pub fn send_event_sync<T: Serialize>(&self, event: &str, data: T) -> bool {
        self.send_event(event, data)
    }

    /// Cancel an existing subscription by key, then store a new token.
    pub fn set_subscription(&self, key: &'static str, resource_id: String, token: CancellationToken) {
        let mut subs = self.subscriptions.lock().unwrap();
        if let Some((_, old_token)) = subs.insert(key, (resource_id, token)) {
            old_token.cancel();
        }
    }

    /// Cancel and remove a subscription by key, but only if the stored
    /// resource_id matches. This prevents a late-arriving unsubscribe
    /// (for container A) from killing a newer subscription (for container B)
    /// created by a concurrent subscribe that raced ahead.
    pub fn cancel_subscription(&self, key: &'static str, resource_id: &str) {
        let mut subs = self.subscriptions.lock().unwrap();
        if let Some((stored_id, _)) = subs.get(key)
            && stored_id == resource_id
        {
            let (_, token) = subs.remove(key).unwrap();
            token.cancel();
        }
    }

    /// Cancel all subscriptions (called on disconnect).
    pub fn cancel_all_subscriptions(&self) {
        let mut subs = self.subscriptions.lock().unwrap();
        for (_, (_, token)) in subs.drain() {
            token.cancel();
        }
    }

    /// Close the connection by sending a Close frame.
    pub fn close(&self) {
        let _ = self.tx.send(Message::Close(None));
    }
}

/// Handler function type for connect events.
pub type ConnectFn = Box<dyn Fn(std::sync::Arc<Conn>) + Send + Sync + 'static>;

/// Handler function type for binary frames.
/// data is [opcode:1][payload:N] (session header already stripped).
pub type BinaryFn =
    Box<dyn Fn(std::sync::Arc<Conn>, u16, &[u8]) + Send + Sync + 'static>;

/// Create a new connection with its write channel.
pub fn new_conn() -> (std::sync::Arc<Conn>, mpsc::UnboundedReceiver<Message>) {
    let id_num = CONN_ID_COUNTER.fetch_add(1, Ordering::Relaxed) + 1;
    let conn_id = format!("c{id_num}");

    let (tx, rx) = mpsc::unbounded_channel();

    let conn = std::sync::Arc::new(Conn {
        id: conn_id,
        user_id: std::sync::atomic::AtomicI32::new(0),
        tx,
        subscriptions: std::sync::Mutex::new(HashMap::new()),
    });

    (conn, rx)
}

/// Single task per connection: owns both halves of the WebSocket and uses a
/// `biased` `select!` loop over three sources:
/// 1. `direct_rx` — queued acks/events from handlers and background tasks
/// 2. `broadcast_rx` — server-wide push events (authenticated only)
/// 3. `ws_read` — incoming client frames, dispatched inline
pub async fn connection_task(
    conn: Arc<Conn>,
    mut direct_rx: mpsc::UnboundedReceiver<Message>,
    mut broadcast_rx: broadcast::Receiver<Arc<str>>,
    mut ws_read: futures_util::stream::SplitStream<WebSocket>,
    mut ws_write: futures_util::stream::SplitSink<WebSocket, Message>,
    server: Arc<super::WsServer>,
) {
    loop {
        tokio::select! {
            biased;

            // Priority 1: send queued responses/events to client
            Some(msg) = direct_rx.recv() => {
                let is_close = matches!(&msg, Message::Close(_));
                if ws_write.send(msg).await.is_err() {
                    break;
                }
                if is_close {
                    debug!(conn = %conn.id, "close frame sent");
                    break;
                }
            }

            // Priority 2: broadcasts (authenticated only)
            result = broadcast_rx.recv() => {
                match result {
                    Ok(payload) if conn.is_authenticated() => {
                        if ws_write.send(Message::Text((*payload).into())).await.is_err() {
                            break;
                        }
                    }
                    Err(broadcast::error::RecvError::Closed) => break,
                    _ => {} // Lagged or not authenticated — skip
                }
            }

            // Priority 3: client frames — dispatch inline
            frame = ws_read.next() => {
                match frame {
                    Some(Ok(Message::Text(text))) => {
                        if let Ok(msg) = serde_json::from_str::<ClientMessage>(&text) {
                            server.dispatch(conn.clone(), msg).await;
                        }
                    }
                    Some(Ok(Message::Binary(data))) if data.len() >= 3 => {
                        let session_id = u16::from_be_bytes([data[0], data[1]]);
                        if let Some(ref handler) = server.binary_handler {
                            handler(conn.clone(), session_id, &data[2..]);
                        }
                    }
                    Some(Ok(Message::Close(_))) | None => break,
                    _ => {}
                }
            }
        }
    }

    // Cleanup
    server.remove(&conn);
    let _ = ws_write.close().await;
}
