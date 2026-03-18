//! Docker event stream consumer + coalesced broadcast dispatcher.
//!
//! Subscribes to Docker events via bollard. Each event:
//! 1. Immediately broadcasts a `resourceEvent` to all authenticated clients
//! 2. Sends to the coalescer, which batches events and triggers filtered
//!    container/resource list broadcasts after a quiet period
//!
//! Coalescing: 50ms quiet window (resets per event), 200ms hard deadline.
//! Tracks resource IDs (not stack names) for precise filtering.

use std::collections::{HashMap, HashSet};
use std::sync::Arc;
use std::time::Duration;

use bollard::models::{EventMessage, EventMessageTypeEnum};
use futures_util::StreamExt;
use serde::Serialize;
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;
use tracing::{debug, warn};

use crate::docker;
use crate::handlers::auth::containers_to_map;
use crate::handlers::AppState;

use super::DispatchMsg;

const QUIET_PERIOD: Duration = Duration::from_millis(50);
const HARD_DEADLINE: Duration = Duration::from_millis(200);
/// If more than this many resources are affected, do a full unfiltered query.
const FILTERED_THRESHOLD: usize = 25;

/// A Docker event converted to our broadcast format.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ResourceEvent {
    #[serde(rename = "type")]
    pub event_type: String,
    pub action: String,
    pub id: String,
    pub name: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub stack_name: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub service_name: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub container_id: String,
}

/// Pending batch of events for the coalescer.
struct PendingBatch {
    /// resource_type → set of resource IDs
    affected: HashMap<String, HashSet<String>>,
    /// resource_type → (resource_name → resource_id) for destroyed resources
    destroyed: HashMap<String, HashMap<String, String>>,
    /// Resource types that need unfiltered full refresh
    force_full: HashSet<String>,
}

impl PendingBatch {
    fn new() -> Self {
        Self {
            affected: HashMap::new(),
            destroyed: HashMap::new(),
            force_full: HashSet::new(),
        }
    }

    fn record_event(&mut self, resource_type: &str, resource_id: &str, resource_name: &str, action: &str) {
        self.affected
            .entry(resource_type.to_string())
            .or_default()
            .insert(resource_id.to_string());

        if action == "destroy" {
            self.destroyed
                .entry(resource_type.to_string())
                .or_default()
                .insert(resource_name.to_string(), resource_id.to_string());
        }
    }

    fn record_full_sync(&mut self, resource_type: &str) {
        self.force_full.insert(resource_type.to_string());
    }
}

/// Spawn the Docker event watcher + coalescer as background tasks.
pub fn spawn(state: Arc<AppState>, dispatch_rx: mpsc::Receiver<DispatchMsg>, cancel: CancellationToken) {
    // Coalescer actor
    let coalescer_state = state.clone();
    let coalescer_cancel = cancel.clone();
    tokio::spawn(async move {
        coalescer_loop(coalescer_state, dispatch_rx, coalescer_cancel).await;
    });

    // Event watcher (reconnect loop)
    tokio::spawn(async move {
        watcher_loop(&state, cancel).await;
    });
}

// ── Watcher ──────────────────────────────────────────────────────────────────

async fn watcher_loop(
    state: &AppState,
    cancel: CancellationToken,
) {
    let mut backoff = Duration::from_secs(1);
    let max_backoff = Duration::from_secs(30);

    loop {
        let start = tokio::time::Instant::now();
        let err = consume_events(state, &cancel).await;

        if cancel.is_cancelled() {
            return;
        }

        if start.elapsed() > Duration::from_secs(30) {
            backoff = Duration::from_secs(1);
        }

        match err {
            Some(e) => warn!("docker events stream error, retrying in {backoff:?}: {e}"),
            None => warn!("docker events stream ended, retrying in {backoff:?}"),
        }

        tokio::select! {
            () = cancel.cancelled() => return,
            () = tokio::time::sleep(backoff) => {},
        }

        backoff = (backoff * 2).min(max_backoff);
    }
}

async fn consume_events(
    state: &AppState,
    cancel: &CancellationToken,
) -> Option<bollard::errors::Error> {
    let mut filters = HashMap::new();
    filters.insert("type".to_string(), vec![
        "container".to_string(),
        "network".to_string(),
        "image".to_string(),
        "volume".to_string(),
    ]);

    let opts = bollard::query_parameters::EventsOptionsBuilder::default()
        .filters(&filters)
        .build();

    let mut stream = state.docker.events(Some(opts));

    loop {
        tokio::select! {
            () = cancel.cancelled() => return None,
            item = stream.next() => {
                match item {
                    Some(Ok(event)) => {
                        handle_event(state, event).await;
                    }
                    Some(Err(e)) => return Some(e),
                    None => return None,
                }
            }
        }
    }
}

/// Process a single Docker event:
/// - Immediately broadcast `resourceEvent` to all authenticated clients
/// - Forward to coalescer for batched list broadcasts
async fn handle_event(
    state: &AppState,
    event: EventMessage,
) {
    let EventMessage { typ, action, actor, .. } = event;
    let action = action.unwrap_or_default();
    let (actor_id, attributes) = match actor {
        Some(a) => (
            a.id.unwrap_or_default(),
            a.attributes.unwrap_or_default(),
        ),
        None => (String::new(), HashMap::new()),
    };
    let event_type = typ.as_ref().map(|t| t.as_ref()).unwrap_or("").to_string();

    let name = attributes.get("name").cloned().unwrap_or_default();

    // Filter container events to relevant actions
    if typ == Some(EventMessageTypeEnum::CONTAINER) {
        match action.as_str() {
            "start" | "stop" | "die" | "pause" | "unpause" | "destroy" | "create" => {}
            a if a.starts_with("health_status") => {}
            _ => return,
        }
    }

    debug!(event_type = %event_type, action = %action, name = %name, "docker event");

    let stack_name = attributes
        .get("com.docker.compose.project")
        .cloned()
        .unwrap_or_default();
    let service_name = attributes
        .get("com.docker.compose.service")
        .cloned()
        .unwrap_or_default();
    let container_id = match typ {
        Some(EventMessageTypeEnum::CONTAINER) => actor_id.clone(),
        Some(EventMessageTypeEnum::NETWORK | EventMessageTypeEnum::VOLUME) => {
            attributes.get("container").cloned().unwrap_or_default()
        }
        _ => String::new(),
    };

    // Publish to internal EventBus (terminals need events even without UI clients)
    state.event_bus.publish(crate::broadcast::eventbus::DockerEvent {
        event_type: event_type.clone(),
        action: action.clone(),
        project: stack_name.clone(),
        service: service_name.clone(),
        container_id: container_id.clone(),
        name: name.clone(),
    });

    // Skip broadcasts if no authenticated clients
    if !state.has_authenticated.load(std::sync::atomic::Ordering::Relaxed) {
        return;
    }

    // 1. Immediate resourceEvent broadcast
    let resource_event = ResourceEvent {
        event_type: event_type.clone(),
        action: action.clone(),
        id: actor_id.clone(),
        name: name.clone(),
        stack_name,
        service_name,
        container_id,
    };
    state.broadcaster.send_event("resourceEvent", &resource_event);

    // 2. Send to coalescer for batched list broadcast
    let _ = state.dispatch_tx.try_send(DispatchMsg::DockerEvent {
        resource_type: event_type,
        resource_id: actor_id,
        resource_name: name,
        action,
    });
}

// ── Coalescer ────────────────────────────────────────────────────────────────

/// Coalescer actor: collects events over a 50ms quiet window (200ms hard
/// deadline), deduplicates by resource type + ID, then triggers filtered
/// list broadcasts.
async fn coalescer_loop(
    state: Arc<AppState>,
    mut rx: mpsc::Receiver<DispatchMsg>,
    cancel: CancellationToken,
) {
    loop {
        // Wait for the first event (or cancellation)
        let first = tokio::select! {
            () = cancel.cancelled() => return,
            evt = rx.recv() => match evt {
                Some(e) => e,
                None => return, // channel closed
            },
        };

        // Start collecting: 50ms quiet, 200ms hard deadline
        let mut batch = PendingBatch::new();
        record_dispatch_msg(&mut batch, first);

        let hard_deadline = tokio::time::Instant::now() + HARD_DEADLINE;

        loop {
            let quiet_timeout = tokio::time::sleep(QUIET_PERIOD);
            let hard_timeout = tokio::time::sleep_until(hard_deadline);

            tokio::select! {
                // Hard deadline: flush regardless
                () = hard_timeout => break,
                // Quiet period expired: flush
                () = quiet_timeout => break,
                // New event: record and reset quiet period
                evt = rx.recv() => {
                    match evt {
                        Some(e) => record_dispatch_msg(&mut batch, e),
                        None => break,
                    }
                }
                () = cancel.cancelled() => return,
            }
        }

        // Dispatch coalesced broadcasts
        dispatch_broadcasts(&state, &batch).await;
    }
}

fn record_dispatch_msg(batch: &mut PendingBatch, msg: DispatchMsg) {
    match msg {
        DispatchMsg::DockerEvent { resource_type, resource_id, resource_name, action } => {
            batch.record_event(&resource_type, &resource_id, &resource_name, &action);
        }
        DispatchMsg::FullSync { resource_type } => {
            batch.record_full_sync(&resource_type);
        }
    }
}

/// Dispatch list broadcasts for affected resource types.
async fn dispatch_broadcasts(state: &AppState, batch: &PendingBatch) {
    for (resource_type, ids) in &batch.affected {
        match resource_type.as_str() {
            "container" => {
                dispatch_container_broadcast(state, ids, batch).await;
            }
            "network" => {
                match docker::network_list(&state.docker).await {
                    Ok(networks) => {
                        let mut map: HashMap<String, serde_json::Value> = networks.into_iter()
                            .map(|n| (n.name.clone(), serde_json::to_value(&n).unwrap_or_default()))
                            .collect();
                        // Insert null for destroyed networks
                        if let Some(destroyed) = batch.destroyed.get("network") {
                            for name in destroyed.keys() {
                                if !map.contains_key(name) {
                                    map.insert(name.clone(), serde_json::Value::Null);
                                }
                            }
                        }
                        state.broadcaster.send_event(
                            "networks",
                            &serde_json::json!({"items": map}),
                        );
                    }
                    Err(e) => warn!("coalescer: network list failed: {e}"),
                }
            }
            "image" => {
                match docker::image_list(&state.docker).await {
                    Ok(images) => {
                        let map: HashMap<String, _> = images.into_iter().map(|i| (i.id.clone(), i)).collect();
                        state.broadcaster.send_event(
                            "images",
                            &serde_json::json!({"items": map}),
                        );
                    }
                    Err(e) => warn!("coalescer: image list failed: {e}"),
                }
            }
            "volume" => {
                match docker::volume_list(&state.docker).await {
                    Ok(volumes) => {
                        let map: HashMap<String, _> = volumes.into_iter().map(|v| (v.name.clone(), v)).collect();
                        state.broadcaster.send_event(
                            "volumes",
                            &serde_json::json!({"items": map}),
                        );
                    }
                    Err(e) => warn!("coalescer: volume list failed: {e}"),
                }
            }
            "stacks" => {
                let stacks = crate::handlers::auth::build_stacks_broadcast(&state.config.stacks_dir);
                state.broadcaster.send_event(
                    "stacks",
                    &serde_json::json!({"items": stacks}),
                );
            }
            _ => {}
        }
    }

    // Also handle force_full types that weren't in affected
    for resource_type in &batch.force_full {
        if batch.affected.contains_key(resource_type) {
            continue; // already dispatched above
        }
        match resource_type.as_str() {
            "container" => {
                dispatch_container_broadcast(state, &HashSet::new(), batch).await;
            }
            "stacks" => {
                let stacks = crate::handlers::auth::build_stacks_broadcast(&state.config.stacks_dir);
                state.broadcaster.send_event(
                    "stacks",
                    &serde_json::json!({"items": stacks}),
                );
            }
            _ => {}
        }
    }
}

/// Dispatch container list broadcast with optional ID filtering.
async fn dispatch_container_broadcast(
    state: &AppState,
    ids: &HashSet<String>,
    batch: &PendingBatch,
) {
    let use_full = batch.force_full.contains("container") || ids.len() > FILTERED_THRESHOLD;

    let containers = if use_full || ids.is_empty() {
        // Full unfiltered query
        docker::container_list(&state.docker, None).await
    } else {
        // Filtered query by IDs
        docker::container_list_by_ids(&state.docker, ids).await
    };

    match containers {
        Ok(containers) => {
            let mut map = containers_to_map(containers);

            // For destroyed containers, insert null entries so the frontend removes them
            if let Some(destroyed) = batch.destroyed.get("container") {
                for name in destroyed.keys() {
                    if !map.contains_key(name) {
                        map.insert(name.clone(), serde_json::Value::Null);
                    }
                }
            }

            if !map.is_empty() {
                state.broadcaster.send_event(
                    "containers",
                    &serde_json::json!({"items": map}),
                );
            }
        }
        Err(e) => warn!("coalescer: container list failed: {e}"),
    }
}
