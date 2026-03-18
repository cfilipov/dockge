pub mod watcher;

use std::sync::Arc;

use serde::Serialize;
use tracing::warn;

use crate::ws::protocol::ServerMessage;

/// Channel-based broadcaster for push events to all connected clients.
/// Uses `tokio::broadcast` — each subscriber (write pump) receives every message.
/// `Arc<str>::clone()` is an Arc increment, so no data is copied per subscriber.
#[derive(Clone)]
pub struct Broadcaster {
    tx: tokio::sync::broadcast::Sender<Arc<str>>,
}

impl Broadcaster {
    pub fn new(capacity: usize) -> Self {
        let (tx, _) = tokio::sync::broadcast::channel(capacity);
        Self { tx }
    }

    /// Get a new receiver for this broadcast channel.
    pub fn subscribe(&self) -> tokio::sync::broadcast::Receiver<Arc<str>> {
        self.tx.subscribe()
    }

    /// Serialize a push event and broadcast to all subscribers.
    pub fn send_event<T: Serialize>(&self, event: &str, data: &T) {
        let msg = ServerMessage {
            event: event.to_string(),
            data,
        };
        match serde_json::to_string(&msg) {
            Ok(json) => {
                let _ = self.tx.send(Arc::from(json));
            }
            Err(e) => {
                warn!("broadcast serialize error: {e}");
            }
        }
    }
}

/// Message sent to the coalescer actor from the Docker event watcher or handlers.
#[allow(dead_code)]
pub enum DispatchMsg {
    /// A Docker event affecting a specific resource.
    DockerEvent {
        resource_type: String,
        resource_id: String,
        resource_name: String,
        action: String,
    },
    /// Request a full unfiltered refresh for a resource type.
    FullSync {
        resource_type: String,
    },
}

/// Control message for the WsServer actor (e.g., disconnect other clients).
pub enum WsControlMsg {
    DisconnectOthers {
        keep_conn_id: String,
        done: tokio::sync::oneshot::Sender<()>,
    },
}
