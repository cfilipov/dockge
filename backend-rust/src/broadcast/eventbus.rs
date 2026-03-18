//! Fan-out Docker events to internal subscribers (terminal reconnection, etc.).
//!
//! Unlike the Broadcaster (which pushes JSON to WebSocket clients), the EventBus
//! distributes structured `DockerEvent`s to in-process subscribers like the
//! combined log merger and single-container log reconnector.

use tokio::sync::broadcast;

/// A Docker event extracted from the event stream, with compose labels resolved.
#[derive(Debug, Clone)]
pub struct DockerEvent {
    pub event_type: String,
    pub action: String,
    pub project: String,
    pub service: String,
    pub container_id: String,
    pub name: String,
}

/// Broadcast channel for internal Docker event subscribers.
#[derive(Clone)]
pub struct EventBus {
    tx: broadcast::Sender<DockerEvent>,
}

impl EventBus {
    pub fn new(capacity: usize) -> Self {
        let (tx, _) = broadcast::channel(capacity);
        Self { tx }
    }

    pub fn subscribe(&self) -> broadcast::Receiver<DockerEvent> {
        self.tx.subscribe()
    }

    pub fn publish(&self, event: DockerEvent) {
        // Ignore send errors — no subscribers is fine
        let _ = self.tx.send(event);
    }
}
