use std::sync::Arc;
use std::time::{Duration, Instant};

use futures_util::StreamExt;
use serde::Serialize;
use tokio_util::sync::CancellationToken;
use tracing::warn;

use crate::docker;
use crate::ws::conn::Conn;
use crate::ws::protocol::{ClientMessage, ErrorResponse};
use crate::ws::WsServer;

use super::{arg_string, parse_args, AppState};

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
    {
        let state = state.clone();
        ws.handle(
            "serviceStatusList",
            move |conn: Arc<Conn>, msg: ClientMessage| {
                let state = state.clone();
                async move {
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
                }
            },
        );
    }

    // getDockerNetworkList
    {
        let state = state.clone();
        ws.handle(
            "getDockerNetworkList",
            move |conn: Arc<Conn>, msg: ClientMessage| {
                let state = state.clone();
                async move {
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
                }
            },
        );
    }

    // getDockerImageList
    {
        let state = state.clone();
        ws.handle(
            "getDockerImageList",
            move |conn: Arc<Conn>, msg: ClientMessage| {
                let state = state.clone();
                async move {
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
                }
            },
        );
    }

    // getDockerVolumeList
    {
        let state = state.clone();
        ws.handle(
            "getDockerVolumeList",
            move |conn: Arc<Conn>, msg: ClientMessage| {
                let state = state.clone();
                async move {
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
                }
            },
        );
    }

    // containerInspect
    {
        let state = state.clone();
        ws.handle(
            "containerInspect",
            move |conn: Arc<Conn>, msg: ClientMessage| {
                let state = state.clone();
                async move {
                    let Some(name) = inspect_extract_arg(&state, &conn, &msg, "Container name").await else { return };
                    let result = state.docker.inspect_container(&name, None::<bollard::query_parameters::InspectContainerOptions>).await;
                    inspect_respond(&conn, &msg, "inspectData", result).await;
                }
            },
        );
    }

    // networkInspect
    {
        let state = state.clone();
        ws.handle(
            "networkInspect",
            move |conn: Arc<Conn>, msg: ClientMessage| {
                let state = state.clone();
                async move {
                    let Some(name) = inspect_extract_arg(&state, &conn, &msg, "Network name").await else { return };
                    let result = state.docker.inspect_network(&name, None::<bollard::query_parameters::InspectNetworkOptions>).await;
                    inspect_respond(&conn, &msg, "networkDetail", result).await;
                }
            },
        );
    }

    // imageInspect
    {
        let state = state.clone();
        ws.handle(
            "imageInspect",
            move |conn: Arc<Conn>, msg: ClientMessage| {
                let state = state.clone();
                async move {
                    let Some(name) = inspect_extract_arg(&state, &conn, &msg, "Image reference").await else { return };
                    let result = state.docker.inspect_image(&name).await;
                    inspect_respond(&conn, &msg, "imageDetail", result).await;
                }
            },
        );
    }

    // volumeInspect
    {
        let state = state.clone();
        ws.handle(
            "volumeInspect",
            move |conn: Arc<Conn>, msg: ClientMessage| {
                let state = state.clone();
                async move {
                    let Some(name) = inspect_extract_arg(&state, &conn, &msg, "Volume name").await else { return };
                    let result = state.docker.inspect_volume(&name).await;
                    inspect_respond(&conn, &msg, "volumeDetail", result).await;
                }
            },
        );
    }

    // subscribeStats — spawn a persistent streaming task
    {
        let state = state.clone();
        ws.handle(
            "subscribeStats",
            move |conn: Arc<Conn>, msg: ClientMessage| {
                let state = state.clone();
                async move {
                    let uid = state.check_login(&conn, &msg).await;
                    if uid == 0 {
                        return;
                    }
                    let args = parse_args(&msg);
                    let container = arg_string(&args, 0);

                    // Create cancellation token and register (cancels any existing stats sub)
                    let token = CancellationToken::new();
                    conn.set_subscription("stats", token.clone());

                    if let Some(id) = msg.id {
                        conn.send_ack(id, serde_json::json!({"ok": true})).await;
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

                                            let mem_usage = stats.memory_stats.as_ref().and_then(|m| m.usage).unwrap_or(0);
                                            let mem_limit = stats.memory_stats.as_ref().and_then(|m| m.limit).unwrap_or(0);

                                            let stats_map = serde_json::json!({
                                                &container: {
                                                    "cpu_percent": cpu_percent,
                                                    "mem_usage": mem_usage,
                                                    "mem_limit": mem_limit,
                                                }
                                            });
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
                }
            },
        );
    }

    // unsubscribeStats
    {
        let state = state.clone();
        ws.handle(
            "unsubscribeStats",
            move |conn: Arc<Conn>, msg: ClientMessage| {
                let state = state.clone();
                async move {
                    let uid = state.check_login(&conn, &msg).await;
                    if uid == 0 {
                        return;
                    }
                    conn.cancel_subscription("stats");
                    if let Some(id) = msg.id {
                        conn.send_ack(id, serde_json::json!({"ok": true})).await;
                    }
                }
            },
        );
    }

    // subscribeTop — spawn a persistent polling task
    {
        let state = state.clone();
        ws.handle(
            "subscribeTop",
            move |conn: Arc<Conn>, msg: ClientMessage| {
                let state = state.clone();
                async move {
                    let uid = state.check_login(&conn, &msg).await;
                    if uid == 0 {
                        return;
                    }
                    let args = parse_args(&msg);
                    let container = arg_string(&args, 0);

                    // Create cancellation token and register (cancels any existing top sub)
                    let token = CancellationToken::new();
                    conn.set_subscription("top", token.clone());

                    if let Some(id) = msg.id {
                        conn.send_ack(id, serde_json::json!({"ok": true})).await;
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
                }
            },
        );
    }

    // unsubscribeTop
    {
        let state = state.clone();
        ws.handle(
            "unsubscribeTop",
            move |conn: Arc<Conn>, msg: ClientMessage| {
                let state = state.clone();
                async move {
                    let uid = state.check_login(&conn, &msg).await;
                    if uid == 0 {
                        return;
                    }
                    conn.cancel_subscription("top");
                    if let Some(id) = msg.id {
                        conn.send_ack(id, serde_json::json!({"ok": true})).await;
                    }
                }
            },
        );
    }
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
async fn push_top(docker: &bollard::Docker, container: &str, conn: &Conn) -> bool {
    match tokio::time::timeout(
        Duration::from_secs(10),
        docker.top_processes(container, None::<bollard::query_parameters::TopOptions>),
    )
    .await
    {
        Ok(Ok(top)) => conn
            .send_event(
                "containerTop",
                serde_json::json!({
                    "ok": true,
                    "processes": top.processes.unwrap_or_default(),
                    "titles": top.titles.unwrap_or_default(),
                }),
            )
            .await,
        Ok(Err(e)) => {
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
        Err(_) => {
            warn!("subscribeTop poll timed out");
            false
        }
    }
}
