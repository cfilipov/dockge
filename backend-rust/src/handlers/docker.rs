use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, Instant};

use futures_util::StreamExt;
use serde::Serialize;
use tokio_util::sync::CancellationToken;
use tracing::warn;

use crate::docker;
use crate::ws::conn::Conn;
use crate::ws::protocol::{ClientMessage, ErrorResponse, OkResponse};
use crate::ws::WsServer;

use super::{arg_string, parse_args, AppState};

/// Pre-formatted container stats matching the frontend's expected shape.
#[derive(Serialize)]
struct ContainerStat {
    #[serde(rename = "Name")]
    name: String,
    #[serde(rename = "CPUPerc")]
    cpu_perc: String,
    #[serde(rename = "MemPerc")]
    mem_perc: String,
    #[serde(rename = "MemUsage")]
    mem_usage: String,
    #[serde(rename = "NetIO")]
    net_io: String,
    #[serde(rename = "BlockIO")]
    block_io: String,
    #[serde(rename = "PIDs")]
    pids: String,
}

/// Format bytes into a human-readable string (e.g. "1.5GiB").
/// Port of Go backend's formatBytes.
fn format_bytes(b: u64) -> String {
    const UNIT: u64 = 1024;
    if b < UNIT {
        return format!("{b}B");
    }
    let mut div = UNIT;
    let mut exp = 0usize;
    let mut n = b / UNIT;
    while n >= UNIT {
        n /= UNIT;
        div *= UNIT;
        exp += 1;
    }
    const UNITS: &[u8] = b"KMGTPE";
    format!("{:.1}{}iB", b as f64 / div as f64, UNITS[exp] as char)
}

fn format_bytes_pair(a: u64, b: u64) -> String {
    format!("{} / {}", format_bytes(a), format_bytes(b))
}

/// Validate login + extract first arg for inspect-style handlers.
/// Returns None if validation fails (ack already sent).
async fn inspect_extract_arg(
    state: &AppState,
    conn: &Conn,
    msg: &ClientMessage,
    arg_label: &str,
) -> Option<String> {
    let uid = state.check_login(conn, msg).await;
    if uid == 0 {
        return None;
    }
    let args = parse_args(msg);
    let name = arg_string(&args, 0);
    if name.is_empty() {
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::new(format!("{arg_label} required")))
                .await;
        }
        return None;
    }
    Some(name)
}

/// Send an inspect result (ok + field) or error ack.
async fn inspect_respond<T: Serialize>(
    conn: &Conn,
    msg: &ClientMessage,
    result_field: &str,
    result: Result<T, bollard::errors::Error>,
) {
    match result {
        Ok(detail) => {
            if let Some(id) = msg.id {
                conn.send_ack(id, serde_json::json!({"ok": true, result_field: detail}))
                    .await;
            }
        }
        Err(e) => {
            warn!("{result_field}: {e}");
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::new(format!("Inspect failed: {e}")))
                    .await;
            }
        }
    }
}

pub fn register(ws: &mut WsServer, state: Arc<AppState>) {
    // serviceStatusList
    ws.handle_with_state("serviceStatusList", state.clone(), |state, conn, msg| async move {
        let uid = state.check_login(&conn, &msg).await;
        if uid == 0 {
            return;
        }
        let args = parse_args(&msg);
        let stack_name = arg_string(&args, 0);

        let containers = docker::container_list(
            &state.docker,
            if stack_name.is_empty() {
                None
            } else {
                Some(&stack_name)
            },
        )
        .await
        .unwrap_or_default();

        // Group by service name
        let mut service_status: std::collections::HashMap<
            String,
            Vec<serde_json::Value>,
        > = std::collections::HashMap::new();
        for c in &containers {
            let key = c.service_name.clone();
            if key.is_empty() {
                continue;
            }
            service_status.entry(key).or_default().push(
                serde_json::json!({
                    "status": c.state,
                    "name": c.name,
                    "image": c.image,
                }),
            );
        }

        if let Some(id) = msg.id {
            conn.send_ack(
                id,
                serde_json::json!({
                    "ok": true,
                    "serviceStatusList": service_status,
                    "serviceUpdateStatus": {},
                    "serviceRecreateStatus": {},
                }),
            )
            .await;
        }
    });

    // getDockerNetworkList
    ws.handle_with_state("getDockerNetworkList", state.clone(), |state, conn, msg| async move {
        let uid = state.check_login(&conn, &msg).await;
        if uid == 0 {
            return;
        }

        let networks = docker::network_list(&state.docker)
            .await
            .unwrap_or_default();
        if let Some(id) = msg.id {
            conn.send_ack(
                id,
                serde_json::json!({
                    "ok": true,
                    "dockerNetworkList": networks,
                }),
            )
            .await;
        }
    });

    // getDockerImageList
    ws.handle_with_state("getDockerImageList", state.clone(), |state, conn, msg| async move {
        let uid = state.check_login(&conn, &msg).await;
        if uid == 0 {
            return;
        }

        let images = docker::image_list(&state.docker)
            .await
            .unwrap_or_default();
        if let Some(id) = msg.id {
            conn.send_ack(
                id,
                serde_json::json!({
                    "ok": true,
                    "dockerImageList": images,
                }),
            )
            .await;
        }
    });

    // getDockerVolumeList
    ws.handle_with_state("getDockerVolumeList", state.clone(), |state, conn, msg| async move {
        let uid = state.check_login(&conn, &msg).await;
        if uid == 0 {
            return;
        }

        let volumes = docker::volume_list(&state.docker)
            .await
            .unwrap_or_default();
        if let Some(id) = msg.id {
            conn.send_ack(
                id,
                serde_json::json!({
                    "ok": true,
                    "dockerVolumeList": volumes,
                }),
            )
            .await;
        }
    });

    // containerInspect
    ws.handle_with_state("containerInspect", state.clone(), |state, conn, msg| async move {
        let Some(name) = inspect_extract_arg(&state, &conn, &msg, "Container name").await else { return };
        let result = state.docker.inspect_container(&name, None::<bollard::query_parameters::InspectContainerOptions>).await;
        inspect_respond(&conn, &msg, "inspectData", result).await;
    });

    // networkInspect
    ws.handle_with_state("networkInspect", state.clone(), |state, conn, msg| async move {
        let Some(name) = inspect_extract_arg(&state, &conn, &msg, "Network name").await else { return };
        let result = docker::network_inspect(&state.docker, &name).await;
        inspect_respond(&conn, &msg, "networkDetail", result).await;
    });

    // imageInspect
    ws.handle_with_state("imageInspect", state.clone(), |state, conn, msg| async move {
        let Some(name) = inspect_extract_arg(&state, &conn, &msg, "Image reference").await else { return };
        let result = docker::image_inspect_detail(&state.docker, &name).await;
        inspect_respond(&conn, &msg, "imageDetail", result).await;
    });

    // volumeInspect
    ws.handle_with_state("volumeInspect", state.clone(), |state, conn, msg| async move {
        let Some(name) = inspect_extract_arg(&state, &conn, &msg, "Volume name").await else { return };
        let result = docker::volume_inspect(&state.docker, &name).await;
        inspect_respond(&conn, &msg, "volumeDetail", result).await;
    });

    // subscribeStats — spawn a persistent streaming task
    ws.handle_with_state("subscribeStats", state.clone(), |state, conn, msg| async move {
        let uid = state.check_login(&conn, &msg).await;
        if uid == 0 {
            return;
        }
        let args = parse_args(&msg);
        let container = arg_string(&args, 0);

        // Create cancellation token and register (cancels any existing stats sub)
        let token = CancellationToken::new();
        conn.set_subscription("stats", container.clone(), token.clone());

        if let Some(id) = msg.id {
            conn.send_ack(id, OkResponse { ok: true, msg: None, token: None }).await;
        }

        // Spawn persistent streaming task
        let conn = conn.clone();
        let docker = state.docker.clone();
        tokio::spawn(async move {
            let opts = bollard::query_parameters::StatsOptionsBuilder::default()
                .stream(true)
                .one_shot(false)
                .build();
            let mut stream = docker.stats(&container, Some(opts));
            let mut last_push = Instant::now() - Duration::from_secs(10); // ensure first frame is sent

            loop {
                tokio::select! {
                    _ = token.cancelled() => return,
                    frame = stream.next() => {
                        match frame {
                            Some(Ok(stats)) => {
                                // Throttle: skip frames within 5s of last push
                                if last_push.elapsed() < Duration::from_secs(5) {
                                    continue;
                                }
                                last_push = Instant::now();

                                // Calculate CPU percent
                                let cpu_percent = calculate_cpu_percent(&stats);

                                // Memory: subtract cache for accurate usage
                                let mem_stats = stats.memory_stats.as_ref();
                                let raw_usage = mem_stats.and_then(|m| m.usage).unwrap_or(0);
                                let cache = mem_stats
                                    .and_then(|m| m.stats.as_ref())
                                    .and_then(|s| s.get("cache").copied())
                                    .unwrap_or(0);
                                let mem_usage = raw_usage.saturating_sub(cache);
                                let mem_limit = mem_stats.and_then(|m| m.limit).unwrap_or(0);

                                // Network I/O: sum across all interfaces
                                let (net_rx, net_tx) = stats
                                    .networks
                                    .as_ref()
                                    .map(|nets| {
                                        nets.values().fold((0u64, 0u64), |(rx, tx), n| {
                                            (
                                                rx + n.rx_bytes.unwrap_or(0),
                                                tx + n.tx_bytes.unwrap_or(0),
                                            )
                                        })
                                    })
                                    .unwrap_or((0, 0));

                                // Block I/O: sum read/write ops
                                let (blk_read, blk_write) = stats
                                    .blkio_stats
                                    .as_ref()
                                    .and_then(|b| b.io_service_bytes_recursive.as_ref())
                                    .map(|entries| {
                                        entries.iter().fold((0u64, 0u64), |(r, w), e| {
                                            match e.op.as_deref() {
                                                Some("read" | "Read") => {
                                                    (r + e.value.unwrap_or(0), w)
                                                }
                                                Some("write" | "Write") => {
                                                    (r, w + e.value.unwrap_or(0))
                                                }
                                                _ => (r, w),
                                            }
                                        })
                                    })
                                    .unwrap_or((0, 0));

                                // PIDs
                                let pids = stats
                                    .pids_stats
                                    .as_ref()
                                    .and_then(|p| p.current)
                                    .unwrap_or(0);

                                let stat = ContainerStat {
                                    name: container.clone(),
                                    cpu_perc: format!("{cpu_percent:.2}%"),
                                    mem_perc: if mem_limit > 0 {
                                        format!(
                                            "{:.2}%",
                                            mem_usage as f64 / mem_limit as f64 * 100.0
                                        )
                                    } else {
                                        "0.00%".into()
                                    },
                                    mem_usage: format_bytes_pair(mem_usage, mem_limit),
                                    net_io: format_bytes_pair(net_rx, net_tx),
                                    block_io: format_bytes_pair(blk_read, blk_write),
                                    pids: pids.to_string(),
                                };

                                let mut stats_map = HashMap::new();
                                stats_map.insert(&container, &stat);

                                let ok = conn.send_event("dockerStats", serde_json::json!({
                                    "ok": true,
                                    "dockerStats": stats_map,
                                })).await;
                                if !ok {
                                    return; // connection dead
                                }
                            }
                            Some(Err(e)) => {
                                warn!("subscribeStats stream error: {e}");
                                return;
                            }
                            None => return, // stream ended
                        }
                    }
                }
            }
        });
    });

    // unsubscribeStats
    ws.handle_with_state("unsubscribeStats", state.clone(), |state, conn, msg| async move {
        let uid = state.check_login(&conn, &msg).await;
        if uid == 0 {
            return;
        }
        let args = parse_args(&msg);
        let container = arg_string(&args, 0);
        conn.cancel_subscription("stats", &container);
        if let Some(id) = msg.id {
            conn.send_ack(id, OkResponse { ok: true, msg: None, token: None }).await;
        }
    });

    // subscribeTop — spawn a persistent polling task
    ws.handle_with_state("subscribeTop", state.clone(), |state, conn, msg| async move {
        let uid = state.check_login(&conn, &msg).await;
        if uid == 0 {
            return;
        }
        let args = parse_args(&msg);
        let container = arg_string(&args, 0);

        // Create cancellation token and register (cancels any existing top sub)
        let token = CancellationToken::new();
        conn.set_subscription("top", container.clone(), token.clone());

        if let Some(id) = msg.id {
            conn.send_ack(id, OkResponse { ok: true, msg: None, token: None }).await;
        }

        // Spawn persistent polling task
        let conn = conn.clone();
        let docker = state.docker.clone();
        tokio::spawn(async move {
            // Push one snapshot immediately
            if !push_top(&docker, &container, &conn).await {
                return;
            }

            let mut interval = tokio::time::interval(Duration::from_secs(10));
            interval.tick().await; // consume the immediate first tick

            loop {
                tokio::select! {
                    _ = token.cancelled() => return,
                    _ = interval.tick() => {
                        if !push_top(&docker, &container, &conn).await {
                            return;
                        }
                    }
                }
            }
        });
    });

    // unsubscribeTop
    ws.handle_with_state("unsubscribeTop", state.clone(), |state, conn, msg| async move {
        let uid = state.check_login(&conn, &msg).await;
        if uid == 0 {
            return;
        }
        let args = parse_args(&msg);
        let container = arg_string(&args, 0);
        conn.cancel_subscription("top", &container);
        if let Some(id) = msg.id {
            conn.send_ack(id, OkResponse { ok: true, msg: None, token: None }).await;
        }
    });
}

/// Calculate CPU usage percentage from stats delta.
fn calculate_cpu_percent(stats: &bollard::models::ContainerStatsResponse) -> f64 {
    let cpu_usage = stats
        .cpu_stats
        .as_ref()
        .and_then(|c| c.cpu_usage.as_ref())
        .and_then(|u| u.total_usage)
        .unwrap_or(0);
    let precpu_usage = stats
        .precpu_stats
        .as_ref()
        .and_then(|c| c.cpu_usage.as_ref())
        .and_then(|u| u.total_usage)
        .unwrap_or(0);

    let system_usage = stats
        .cpu_stats
        .as_ref()
        .and_then(|c| c.system_cpu_usage)
        .unwrap_or(0);
    let presystem_usage = stats
        .precpu_stats
        .as_ref()
        .and_then(|c| c.system_cpu_usage)
        .unwrap_or(0);

    let online_cpus = stats
        .cpu_stats
        .as_ref()
        .and_then(|c| c.online_cpus)
        .unwrap_or(1);

    let cpu_delta = cpu_usage as i64 - precpu_usage as i64;
    let system_delta = system_usage as i64 - presystem_usage as i64;

    if system_delta > 0 && cpu_delta > 0 {
        (cpu_delta as f64 / system_delta as f64) * online_cpus as f64 * 100.0
    } else {
        0.0
    }
}

/// Push a single top-processes snapshot. Returns false if the connection is dead or the container is gone.
async fn push_top(docker: &crate::docker::DockerClient, container: &str, conn: &Conn) -> bool {
    match docker
        .top_processes(container, None::<bollard::query_parameters::TopOptions>)
        .await
    {
        Ok(top) => conn
            .send_event(
                "containerTop",
                serde_json::json!({
                    "ok": true,
                    "processes": top.processes.unwrap_or_default(),
                    "titles": top.titles.unwrap_or_default(),
                }),
            )
            .await,
        Err(e) => {
            warn!("subscribeTop poll error: {e}");
            let _ = conn
                .send_event(
                    "containerTop",
                    serde_json::json!({
                        "ok": true,
                        "processes": [],
                        "titles": [],
                    }),
                )
                .await;
            false
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn format_bytes_zero() {
        assert_eq!(format_bytes(0), "0B");
    }

    #[test]
    fn format_bytes_below_unit() {
        assert_eq!(format_bytes(1023), "1023B");
    }

    #[test]
    fn format_bytes_exact_kib() {
        assert_eq!(format_bytes(1024), "1.0KiB");
    }

    #[test]
    fn format_bytes_fractional_kib() {
        assert_eq!(format_bytes(1536), "1.5KiB");
    }

    #[test]
    fn format_bytes_exact_mib() {
        assert_eq!(format_bytes(1_048_576), "1.0MiB");
    }

    #[test]
    fn format_bytes_fractional_gib() {
        assert_eq!(format_bytes(1_610_612_736), "1.5GiB");
    }

    #[test]
    fn format_bytes_exact_tib() {
        assert_eq!(format_bytes(1_099_511_627_776), "1.0TiB");
    }

    #[test]
    fn format_bytes_pair_output() {
        assert_eq!(
            format_bytes_pair(1_048_576, 512),
            "1.0MiB / 512B"
        );
    }
}
