//! Image update checking: compares local vs remote digests for compose services.
//!
//! Background checker runs on a configurable interval (default 6h), storing
//! results in redb. Provides `checkImageUpdates` handler for on-demand checks.

use std::sync::Arc;
use std::time::Duration;

use tokio_util::sync::CancellationToken;
use tracing::{debug, info, warn};

use crate::db;
use crate::ws::conn::Conn;
use crate::ws::protocol::ClientMessage;
use crate::ws::WsServer;

use super::{arg_string, parse_args, AppState};

const DEFAULT_CHECK_INTERVAL: Duration = Duration::from_secs(6 * 3600);
const IMAGE_CHECK_CONCURRENCY: usize = 3;
const PER_IMAGE_TIMEOUT: Duration = Duration::from_secs(30);

pub fn register(ws: &mut WsServer, state: Arc<AppState>) {
    let state_clone = state.clone();
    ws.handle(
        "checkImageUpdates",
        move |conn: Arc<Conn>, msg: ClientMessage| {
            let state = state_clone.clone();
            async move {
                let uid = state.check_login(&conn, &msg).await;
                if uid == 0 {
                    return;
                }

                let args = parse_args(&msg);
                let stack_name = arg_string(&args, 0);

                // Ack immediately
                if let Some(id) = msg.id {
                    conn.send_ack(
                        id,
                        serde_json::json!({"ok": true, "updated": true}),
                    )
                    .await;
                }

                if !stack_name.is_empty() {
                    let state = state.clone();
                    tokio::spawn(async move {
                        check_image_updates_for_stack(&state, &stack_name).await;
                        trigger_updates_broadcast(&state);
                    });
                }
            }
        },
    );
}

/// Spawn the background image update checker.
pub fn spawn_checker(state: Arc<AppState>, cancel: CancellationToken) {
    tokio::spawn(async move {
        // Short delay on startup
        tokio::select! {
            () = cancel.cancelled() => return,
            () = tokio::time::sleep(Duration::from_secs(5)) => {},
        }

        let interval = get_check_interval(&state);

        // Check if enough time has elapsed since last check
        let last_check = get_last_check_time(&state);
        let elapsed = std::time::SystemTime::now()
            .duration_since(last_check)
            .unwrap_or(interval);

        if elapsed < interval {
            let remaining = interval - elapsed;
            debug!("image update checker: deferring first check by {remaining:?}");
            tokio::select! {
                () = cancel.cancelled() => return,
                () = tokio::time::sleep(remaining) => {},
            }
        }

        // First check
        if is_check_enabled(&state) {
            check_all_image_updates(&state).await;
            set_last_check_time(&state);
            trigger_updates_broadcast(&state);
        }

        // Periodic loop
        loop {
            let interval = get_check_interval(&state);
            tokio::select! {
                () = cancel.cancelled() => return,
                () = tokio::time::sleep(interval) => {},
            }

            if is_check_enabled(&state) {
                check_all_image_updates(&state).await;
                set_last_check_time(&state);
                trigger_updates_broadcast(&state);
            }
        }
    });
}

/// Check all stacks for image updates.
async fn check_all_image_updates(state: &AppState) {
    let entries = match std::fs::read_dir(&state.config.stacks_dir) {
        Ok(e) => e,
        Err(e) => {
            warn!("check_all_image_updates: read stacks dir: {e}");
            return;
        }
    };

    let mut stack_names = Vec::new();
    for entry in entries.flatten() {
        if !entry.path().is_dir() {
            continue;
        }
        let name = entry.file_name().to_string_lossy().to_string();
        if name.starts_with('.') {
            continue;
        }
        // Check if it has a compose file
        let compose_path = entry.path().join("compose.yaml");
        if compose_path.exists() {
            stack_names.push(name);
        }
    }

    if stack_names.is_empty() {
        return;
    }

    info!(
        "background image update check starting ({} stacks)",
        stack_names.len()
    );

    let sem = Arc::new(tokio::sync::Semaphore::new(IMAGE_CHECK_CONCURRENCY));
    let mut handles = Vec::new();

    for name in stack_names {
        let sem = sem.clone();
        let state = state as *const AppState as usize;
        handles.push(tokio::spawn(async move {
            let _permit = sem.acquire().await.unwrap();
            // Safety: AppState outlives this task
            let state = unsafe { &*(state as *const AppState) };
            check_image_updates_for_stack(state, &name).await;
        }));
    }

    for handle in handles {
        let _ = handle.await;
    }

    debug!("background image update check complete");
}

/// Check a single stack for image updates.
async fn check_image_updates_for_stack(state: &AppState, stack_name: &str) {
    let compose_path = format!("{}/{}/compose.yaml", state.config.stacks_dir, stack_name);
    let yaml = match std::fs::read_to_string(&compose_path) {
        Ok(y) => y,
        Err(_) => return,
    };

    // Simple YAML parsing: extract image references from services
    let images = parse_service_images(&yaml);
    if images.is_empty() {
        return;
    }

    for (service_name, image_ref) in &images {
        let ctx_cancel = CancellationToken::new();
        let timeout = tokio::time::timeout(PER_IMAGE_TIMEOUT, async {
            let local = image_digest(state, image_ref).await;
            let remote = manifest_digest(state, image_ref).await;
            (local, remote)
        });

        let (local, remote) = match timeout.await {
            Ok((l, r)) => (l, r),
            Err(_) => {
                debug!(stack = %stack_name, service = %service_name, "image check timed out");
                (String::new(), String::new())
            }
        };
        drop(ctx_cancel);

        let has_update = !local.is_empty() && !remote.is_empty() && local != remote;
        let check_status = if local.is_empty() || remote.is_empty() {
            "failed"
        } else {
            "ok"
        };

        // Store in redb
        let key = format!("{}/{}", stack_name, service_name);
        let value = serde_json::json!({
            "stackName": stack_name,
            "serviceName": service_name,
            "imageRef": image_ref,
            "localDigest": local,
            "remoteDigest": remote,
            "hasUpdate": has_update,
            "checkStatus": check_status,
        });

        if let Ok(write_txn) = state.db.begin_write() {
            if let Ok(mut table) = write_txn.open_table(db::IMAGE_UPDATES_TABLE) {
                let _ = table.insert(key.as_str(), value.to_string().as_str());
            }
            let _ = write_txn.commit();
        }
    }
}

/// Get local image digest via Docker inspect.
async fn image_digest(state: &AppState, image_ref: &str) -> String {
    match state
        .docker
        .inspect_image(image_ref)
        .await
    {
        Ok(info) => {
            if let Some(digests) = info.repo_digests {
                for d in &digests {
                    if let Some(idx) = d.find('@') {
                        return d[idx + 1..].to_string();
                    }
                }
                digests.first().cloned().unwrap_or_default()
            } else {
                String::new()
            }
        }
        Err(_) => String::new(),
    }
}

/// Get remote manifest digest via distribution inspect.
async fn manifest_digest(state: &AppState, image_ref: &str) -> String {
    match state
        .docker
        .inspect_registry_image(
            image_ref,
            None::<bollard::auth::DockerCredentials>,
        )
        .await
    {
        Ok(info) => info
            .descriptor
            .digest
            .unwrap_or_default(),
        Err(_) => String::new(),
    }
}

/// Simple YAML parser to extract service image references.
/// Returns Vec<(service_name, image_ref)>.
fn parse_service_images(yaml: &str) -> Vec<(String, String)> {
    let mut results = Vec::new();
    let mut in_services = false;
    let mut current_service: Option<String> = None;

    for line in yaml.lines() {
        let trimmed = line.trim();

        // Detect "services:" top-level key
        if !line.starts_with(' ') && !line.starts_with('\t') {
            in_services = trimmed == "services:" || trimmed.starts_with("services:");
            current_service = None;
            continue;
        }

        if !in_services {
            continue;
        }

        // 2-space or tab indented = service name
        let indent = line.len() - line.trim_start().len();
        if indent <= 4 && trimmed.ends_with(':') && !trimmed.starts_with('#') {
            current_service = Some(trimmed.trim_end_matches(':').to_string());
            continue;
        }

        // 4+ space indented = service property
        if current_service.is_some() && trimmed.starts_with("image:") {
            let image = trimmed
                .strip_prefix("image:")
                .unwrap_or("")
                .trim()
                .trim_matches('"')
                .trim_matches('\'')
                .to_string();
            if !image.is_empty() {
                results.push((current_service.clone().unwrap(), image));
            }
        }
    }

    results
}

/// Broadcast image update results to all clients.
fn trigger_updates_broadcast(state: &AppState) {
    let mut updates = Vec::new();

    if let Ok(read_txn) = state.db.begin_read()
        && let Ok(table) = read_txn.open_table(db::IMAGE_UPDATES_TABLE)
    {
        use redb::ReadableTable;
        if let Ok(iter) = table.iter() {
            for entry in iter.flatten() {
                if let Ok(value) = serde_json::from_str::<serde_json::Value>(entry.1.value()) {
                    updates.push(value);
                }
            }
        }
    }

    state.broadcaster.send_event(
        "updates",
        &serde_json::json!({"items": updates}),
    );
}

fn get_check_interval(state: &AppState) -> Duration {
    let settings = state.get_all_settings().unwrap_or_default();
    settings
        .get("imageUpdateCheckInterval")
        .and_then(|v| v.parse::<f64>().ok())
        .filter(|h| *h > 0.0)
        .map(|h| Duration::from_secs_f64(h * 3600.0))
        .unwrap_or(DEFAULT_CHECK_INTERVAL)
}

fn is_check_enabled(state: &AppState) -> bool {
    let settings = state.get_all_settings().unwrap_or_default();
    settings
        .get("imageUpdateCheckEnabled")
        .map(|v| v != "0" && v != "false")
        .unwrap_or(true)
}

fn get_last_check_time(state: &AppState) -> std::time::SystemTime {
    let settings = state.get_all_settings().unwrap_or_default();
    settings
        .get("imageUpdateLastCheck")
        .and_then(|v| v.parse::<u64>().ok())
        .map(|ts| std::time::UNIX_EPOCH + Duration::from_secs(ts))
        .unwrap_or(std::time::UNIX_EPOCH)
}

fn set_last_check_time(state: &AppState) {
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs();
    if let Err(e) = state.set_setting("imageUpdateLastCheck", &now.to_string()) {
        tracing::warn!("failed to save imageUpdateLastCheck: {e}");
    }
}
