//! File system watcher for compose files in the stacks directory.
//!
//! Watches each stack subdirectory for compose file changes and triggers
//! debounced `FullSync { resource_type: "stacks" }` broadcasts through the
//! coalescer so the frontend sees updated stack metadata.

use std::collections::HashMap;
use std::path::{Path, PathBuf};
use std::time::Duration;

use notify::{EventKind, RecommendedWatcher, RecursiveMode, Watcher};
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;
use tracing::{debug, info, warn};

use crate::broadcast::DispatchMsg;

const DEBOUNCE_MS: u64 = 200;

const COMPOSE_FILENAMES: &[&str] = &[
    "compose.yaml",
    "compose.yml",
    "docker-compose.yaml",
    "docker-compose.yml",
];

/// Spawn the compose file watcher as a background task.
pub fn spawn(
    stacks_dir: String,
    dispatch_tx: mpsc::Sender<DispatchMsg>,
    cancel: CancellationToken,
) {
    tokio::spawn(async move {
        watcher_loop(&stacks_dir, &dispatch_tx, &cancel).await;
    });
}

async fn watcher_loop(
    stacks_dir: &str,
    dispatch_tx: &mpsc::Sender<DispatchMsg>,
    cancel: &CancellationToken,
) {
    let mut backoff = Duration::from_secs(1);
    let max_backoff = Duration::from_secs(30);

    loop {
        let start = tokio::time::Instant::now();
        let err = run_watcher(stacks_dir, dispatch_tx, cancel).await;

        if cancel.is_cancelled() {
            return;
        }

        if start.elapsed() > Duration::from_secs(30) {
            backoff = Duration::from_secs(1);
        }

        match err {
            Some(e) => warn!("compose file watcher error, retrying in {backoff:?}: {e}"),
            None => warn!("compose file watcher ended, retrying in {backoff:?}"),
        }

        tokio::select! {
            () = cancel.cancelled() => return,
            () = tokio::time::sleep(backoff) => {},
        }

        backoff = (backoff * 2).min(max_backoff);
    }
}

async fn run_watcher(
    stacks_dir: &str,
    dispatch_tx: &mpsc::Sender<DispatchMsg>,
    cancel: &CancellationToken,
) -> Option<String> {
    let stacks_path = PathBuf::from(stacks_dir);
    if !stacks_path.is_dir() {
        return Some(format!("stacks dir does not exist: {stacks_dir}"));
    }

    let (fs_tx, mut fs_rx) = mpsc::channel::<notify::Event>(256);

    let mut watcher = match RecommendedWatcher::new(
        move |res: Result<notify::Event, notify::Error>| {
            if let Ok(event) = res {
                let _ = fs_tx.blocking_send(event);
            }
        },
        notify::Config::default(),
    ) {
        Ok(w) => w,
        Err(e) => return Some(format!("failed to create watcher: {e}")),
    };

    // Watch the stacks directory itself (non-recursive — we watch subdirs individually)
    if let Err(e) = watcher.watch(&stacks_path, RecursiveMode::NonRecursive) {
        return Some(format!("failed to watch stacks dir: {e}"));
    }

    // Watch each existing subdirectory
    if let Ok(entries) = std::fs::read_dir(&stacks_path) {
        for entry in entries.flatten() {
            let path = entry.path();
            if path.is_dir() && !entry.file_name().to_string_lossy().starts_with('.') {
                let _ = watcher.watch(&path, RecursiveMode::NonRecursive);
            }
        }
    }

    info!(stacks_dir = %stacks_dir, "compose file watcher started");

    // Debounce: track pending fire times per stack
    let mut pending: HashMap<String, tokio::time::Instant> = HashMap::new();

    loop {
        // Calculate next fire time
        let next_fire = pending.values().min().copied();
        let sleep_fut = match next_fire {
            Some(t) => tokio::time::sleep_until(t),
            None => tokio::time::sleep(Duration::from_secs(3600)), // effectively infinite
        };

        tokio::select! {
            () = cancel.cancelled() => return None,
            () = sleep_fut, if next_fire.is_some() => {
                // Fire expired timers
                let now = tokio::time::Instant::now();
                let expired: Vec<String> = pending
                    .iter()
                    .filter(|(_, t)| **t <= now)
                    .map(|(k, _)| k.clone())
                    .collect();

                if !expired.is_empty() {
                    for key in &expired {
                        pending.remove(key);
                    }
                    debug!(stacks = ?expired, "compose file change detected, dispatching stacks sync");
                    let _ = dispatch_tx.try_send(DispatchMsg::FullSync {
                        resource_type: "stacks".to_string(),
                    });
                }
            }
            event = fs_rx.recv() => {
                let event = match event {
                    Some(e) => e,
                    None => return Some("fs event channel closed".to_string()),
                };

                // Only react to content changes, creation, removal, and renames.
                // Ignore metadata/chmod/access events to prevent a feedback loop
                // where reading compose files triggers metadata inotify events on
                // ZFS/CoW filesystems, causing endless stacks broadcasts.
                if !matches!(
                    event.kind,
                    EventKind::Create(_)
                        | EventKind::Modify(notify::event::ModifyKind::Data(_))
                        | EventKind::Modify(notify::event::ModifyKind::Name(_))
                        | EventKind::Remove(_)
                ) {
                    continue;
                }

                for path in &event.paths {
                    // New subdirectory created in stacks dir → start watching it
                    if matches!(event.kind, EventKind::Create(_))
                        && path.parent() == Some(&stacks_path)
                        && path.is_dir()
                    {
                        let _ = watcher.watch(path, RecursiveMode::NonRecursive);
                        // Also trigger a stacks sync for the new directory
                        if let Some(name) = path.file_name().and_then(|n| n.to_str()) {
                            pending.insert(
                                name.to_string(),
                                tokio::time::Instant::now() + Duration::from_millis(DEBOUNCE_MS),
                            );
                        }
                        continue;
                    }

                    // Compose file change in a stack subdirectory
                    if is_compose_file(path)
                        && let Some(stack_name) = extract_stack_name(path, &stacks_path)
                    {
                        pending.insert(
                            stack_name,
                            tokio::time::Instant::now() + Duration::from_millis(DEBOUNCE_MS),
                        );
                    }
                }
            }
        }
    }
}

fn is_compose_file(path: &Path) -> bool {
    path.file_name()
        .and_then(|n| n.to_str())
        .is_some_and(|name| COMPOSE_FILENAMES.contains(&name))
}

fn extract_stack_name(path: &Path, stacks_dir: &Path) -> Option<String> {
    let parent = path.parent()?;
    if parent.parent()? == stacks_dir {
        parent
            .file_name()
            .and_then(|n| n.to_str())
            .map(|s| s.to_string())
    } else if parent == stacks_dir {
        // Direct file in stacks dir (rare) — use filename as key
        path.file_stem()
            .and_then(|n| n.to_str())
            .map(|s| s.to_string())
    } else {
        None
    }
}
