use std::sync::Arc;

use tracing::warn;

use crate::docker;
use crate::ws::conn::Conn;
use crate::ws::protocol::{ClientMessage, ErrorResponse};
use crate::ws::WsServer;

use super::{parse_args, arg_string, AppState};

pub fn register(ws: &mut WsServer, state: Arc<AppState>) {
    // serviceStatusList
    {
        let state = state.clone();
        ws.handle("serviceStatusList", move |conn: Arc<Conn>, msg: ClientMessage| {
            let state = state.clone();
            async move {
                let uid = state.check_login(&conn, &msg).await;
                if uid == 0 { return; }
                let args = parse_args(&msg);
                let stack_name = arg_string(&args, 0);

                let containers = docker::container_list(&state.docker, if stack_name.is_empty() { None } else { Some(&stack_name) })
                    .await
                    .unwrap_or_default();

                // Group by service name
                let mut service_status: std::collections::HashMap<String, Vec<serde_json::Value>> = std::collections::HashMap::new();
                for c in &containers {
                    let key = c.service_name.clone();
                    if key.is_empty() { continue; }
                    service_status.entry(key).or_default().push(serde_json::json!({
                        "status": c.state,
                        "name": c.name,
                        "image": c.image,
                    }));
                }

                if let Some(id) = msg.id {
                    conn.send_ack(id, serde_json::json!({
                        "ok": true,
                        "serviceStatusList": service_status,
                        "serviceUpdateStatus": {},
                        "serviceRecreateStatus": {},
                    })).await;
                }
            }
        });
    }

    // getDockerNetworkList
    {
        let state = state.clone();
        ws.handle("getDockerNetworkList", move |conn: Arc<Conn>, msg: ClientMessage| {
            let state = state.clone();
            async move {
                let uid = state.check_login(&conn, &msg).await;
                if uid == 0 { return; }

                let networks = docker::network_list(&state.docker).await.unwrap_or_default();
                if let Some(id) = msg.id {
                    conn.send_ack(id, serde_json::json!({
                        "ok": true,
                        "dockerNetworkList": networks,
                    })).await;
                }
            }
        });
    }

    // getDockerImageList
    {
        let state = state.clone();
        ws.handle("getDockerImageList", move |conn: Arc<Conn>, msg: ClientMessage| {
            let state = state.clone();
            async move {
                let uid = state.check_login(&conn, &msg).await;
                if uid == 0 { return; }

                let images = docker::image_list(&state.docker).await.unwrap_or_default();
                if let Some(id) = msg.id {
                    conn.send_ack(id, serde_json::json!({
                        "ok": true,
                        "dockerImageList": images,
                    })).await;
                }
            }
        });
    }

    // getDockerVolumeList
    {
        let state = state.clone();
        ws.handle("getDockerVolumeList", move |conn: Arc<Conn>, msg: ClientMessage| {
            let state = state.clone();
            async move {
                let uid = state.check_login(&conn, &msg).await;
                if uid == 0 { return; }

                let volumes = docker::volume_list(&state.docker).await.unwrap_or_default();
                if let Some(id) = msg.id {
                    conn.send_ack(id, serde_json::json!({
                        "ok": true,
                        "dockerVolumeList": volumes,
                    })).await;
                }
            }
        });
    }

    // containerInspect
    {
        let state = state.clone();
        ws.handle("containerInspect", move |conn: Arc<Conn>, msg: ClientMessage| {
            let state = state.clone();
            async move {
                let uid = state.check_login(&conn, &msg).await;
                if uid == 0 { return; }

                let args = parse_args(&msg);
                let container_name = arg_string(&args, 0);
                if container_name.is_empty() {
                    if let Some(id) = msg.id {
                        conn.send_ack(id, ErrorResponse::new("Container name required")).await;
                    }
                    return;
                }

                match state.docker.inspect_container(&container_name, None::<bollard::query_parameters::InspectContainerOptions>).await {
                    Ok(inspect) => {
                        if let Some(id) = msg.id {
                            conn.send_ack(id, serde_json::json!({
                                "ok": true,
                                "inspectData": inspect,
                            })).await;
                        }
                    }
                    Err(e) => {
                        warn!("containerInspect: {e}");
                        if let Some(id) = msg.id {
                            conn.send_ack(id, ErrorResponse::new(format!("Inspect failed: {e}"))).await;
                        }
                    }
                }
            }
        });
    }

    // networkInspect
    {
        let state = state.clone();
        ws.handle("networkInspect", move |conn: Arc<Conn>, msg: ClientMessage| {
            let state = state.clone();
            async move {
                let uid = state.check_login(&conn, &msg).await;
                if uid == 0 { return; }

                let args = parse_args(&msg);
                let network_name = arg_string(&args, 0);
                if network_name.is_empty() {
                    if let Some(id) = msg.id {
                        conn.send_ack(id, ErrorResponse::new("Network name required")).await;
                    }
                    return;
                }

                match state.docker.inspect_network(&network_name, None::<bollard::query_parameters::InspectNetworkOptions>).await {
                    Ok(detail) => {
                        if let Some(id) = msg.id {
                            conn.send_ack(id, serde_json::json!({
                                "ok": true,
                                "networkDetail": detail,
                            })).await;
                        }
                    }
                    Err(e) => {
                        warn!("networkInspect: {e}");
                        if let Some(id) = msg.id {
                            conn.send_ack(id, ErrorResponse::new(format!("Inspect failed: {e}"))).await;
                        }
                    }
                }
            }
        });
    }

    // imageInspect
    {
        let state = state.clone();
        ws.handle("imageInspect", move |conn: Arc<Conn>, msg: ClientMessage| {
            let state = state.clone();
            async move {
                let uid = state.check_login(&conn, &msg).await;
                if uid == 0 { return; }

                let args = parse_args(&msg);
                let image_ref = arg_string(&args, 0);
                if image_ref.is_empty() {
                    if let Some(id) = msg.id {
                        conn.send_ack(id, ErrorResponse::new("Image reference required")).await;
                    }
                    return;
                }

                match state.docker.inspect_image(&image_ref).await {
                    Ok(detail) => {
                        if let Some(id) = msg.id {
                            conn.send_ack(id, serde_json::json!({
                                "ok": true,
                                "imageDetail": detail,
                            })).await;
                        }
                    }
                    Err(e) => {
                        warn!("imageInspect: {e}");
                        if let Some(id) = msg.id {
                            conn.send_ack(id, ErrorResponse::new(format!("Inspect failed: {e}"))).await;
                        }
                    }
                }
            }
        });
    }

    // volumeInspect
    {
        let state = state.clone();
        ws.handle("volumeInspect", move |conn: Arc<Conn>, msg: ClientMessage| {
            let state = state.clone();
            async move {
                let uid = state.check_login(&conn, &msg).await;
                if uid == 0 { return; }

                let args = parse_args(&msg);
                let volume_name = arg_string(&args, 0);
                if volume_name.is_empty() {
                    if let Some(id) = msg.id {
                        conn.send_ack(id, ErrorResponse::new("Volume name required")).await;
                    }
                    return;
                }

                match state.docker.inspect_volume(&volume_name).await {
                    Ok(detail) => {
                        if let Some(id) = msg.id {
                            conn.send_ack(id, serde_json::json!({
                                "ok": true,
                                "volumeDetail": detail,
                            })).await;
                        }
                    }
                    Err(e) => {
                        warn!("volumeInspect: {e}");
                        if let Some(id) = msg.id {
                            conn.send_ack(id, ErrorResponse::new(format!("Inspect failed: {e}"))).await;
                        }
                    }
                }
            }
        });
    }

    // subscribeStats — spawn a task that pushes stats periodically
    {
        let state = state.clone();
        ws.handle("subscribeStats", move |conn: Arc<Conn>, msg: ClientMessage| {
            let state = state.clone();
            async move {
                let uid = state.check_login(&conn, &msg).await;
                if uid == 0 { return; }
                let args = parse_args(&msg);
                let container = arg_string(&args, 0);

                if let Some(id) = msg.id {
                    conn.send_ack(id, serde_json::json!({"ok": true})).await;
                }

                // Push one stats snapshot immediately
                use futures_util::StreamExt;
                let opts = bollard::query_parameters::StatsOptionsBuilder::default()
                    .stream(false)
                    .one_shot(true)
                    .build();
                let mut stream = state.docker.stats(&container, Some(opts));
                if let Some(Ok(stats)) = stream.next().await {
                    let mem_usage = stats.memory_stats.as_ref().and_then(|m| m.usage).unwrap_or(0);
                    let mem_limit = stats.memory_stats.as_ref().and_then(|m| m.limit).unwrap_or(0);
                    let stats_map = serde_json::json!({
                        &container: {
                            "cpu_percent": 0.0,
                            "mem_usage": mem_usage,
                            "mem_limit": mem_limit,
                        }
                    });
                    conn.send_event("dockerStats", serde_json::json!({
                        "ok": true,
                        "dockerStats": stats_map,
                    })).await;
                }
            }
        });
    }

    // unsubscribeStats
    {
        let state = state.clone();
        ws.handle("unsubscribeStats", move |conn: Arc<Conn>, msg: ClientMessage| {
            let state = state.clone();
            async move {
                let uid = state.check_login(&conn, &msg).await;
                if uid == 0 { return; }
                if let Some(id) = msg.id {
                    conn.send_ack(id, serde_json::json!({"ok": true})).await;
                }
            }
        });
    }

    // subscribeTop — push process list
    {
        let state = state.clone();
        ws.handle("subscribeTop", move |conn: Arc<Conn>, msg: ClientMessage| {
            let state = state.clone();
            async move {
                let uid = state.check_login(&conn, &msg).await;
                if uid == 0 { return; }
                let args = parse_args(&msg);
                let container = arg_string(&args, 0);

                if let Some(id) = msg.id {
                    conn.send_ack(id, serde_json::json!({"ok": true})).await;
                }

                // Push one top snapshot
                match state.docker.top_processes(&container, None::<bollard::query_parameters::TopOptions>).await {
                    Ok(top) => {
                        conn.send_event("containerTop", serde_json::json!({
                            "ok": true,
                            "processes": top.processes.unwrap_or_default(),
                            "titles": top.titles.unwrap_or_default(),
                        })).await;
                    }
                    Err(e) => {
                        warn!("subscribeTop: {e}");
                    }
                }
            }
        });
    }

    // unsubscribeTop
    {
        let state = state.clone();
        ws.handle("unsubscribeTop", move |conn: Arc<Conn>, msg: ClientMessage| {
            let state = state.clone();
            async move {
                let uid = state.check_login(&conn, &msg).await;
                if uid == 0 { return; }
                if let Some(id) = msg.id {
                    conn.send_ack(id, serde_json::json!({"ok": true})).await;
                }
            }
        });
    }
}
