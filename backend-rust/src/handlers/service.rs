use std::sync::Arc;

use tracing::warn;

use crate::docker;
use crate::ws::conn::Conn;
use crate::ws::protocol::{ClientMessage, ErrorResponse, OkResponse};
use crate::ws::WsServer;

use super::{arg_string, parse_args, AppState};

pub fn register(ws: &mut WsServer, state: Arc<AppState>) {
    // Per-service lifecycle: resolve container name from stack+service, then call Docker SDK
    for (event, docker_action) in &[
        ("startService", DockerAction::Start),
        ("stopService", DockerAction::Stop),
        ("restartService", DockerAction::Restart),
        ("recreateService", DockerAction::Restart), // recreate = restart for now
        ("updateService", DockerAction::Restart),   // update = pull + restart
    ] {
        let state = state.clone();
        let action = *docker_action;
        ws.handle(event, move |conn: Arc<Conn>, msg: ClientMessage| {
            let state = state.clone();
            async move {
                let uid = state.check_login(&conn, &msg).await;
                if uid == 0 {
                    return;
                }

                let args = parse_args(&msg);
                let stack_name = arg_string(&args, 0);
                let service_name = arg_string(&args, 1);
                if stack_name.is_empty() || service_name.is_empty() {
                    if let Some(id) = msg.id {
                        conn.send_ack(
                            id,
                            ErrorResponse::new("Stack name and service name required"),
                        )
                        .await;
                    }
                    return;
                }

                // Find the container for this stack+service
                let container_name = format!("{}-{}-1", stack_name, service_name);

                // Ack immediately
                if let Some(id) = msg.id {
                    conn.send_ack(id, OkResponse { ok: true, msg: None, token: None }).await;
                }

                // Execute Docker action with timeout
                let result = match action {
                    DockerAction::Start => {
                        docker::with_timeout(state.docker.start_container(
                            &container_name,
                            None::<bollard::query_parameters::StartContainerOptions>,
                        ))
                        .await
                    }
                    DockerAction::Stop => {
                        docker::with_timeout(state.docker.stop_container(
                            &container_name,
                            None::<bollard::query_parameters::StopContainerOptions>,
                        ))
                        .await
                    }
                    DockerAction::Restart => {
                        docker::with_timeout(state.docker.restart_container(
                            &container_name,
                            None::<bollard::query_parameters::RestartContainerOptions>,
                        ))
                        .await
                    }
                };

                if let Err(e) = result {
                    warn!(container = %container_name, "service action failed: {e}");
                }
            }
        });
    }

    // Per-container lifecycle
    for (event, docker_action) in &[
        ("startContainer", DockerAction::Start),
        ("stopContainer", DockerAction::Stop),
        ("restartContainer", DockerAction::Restart),
    ] {
        let state = state.clone();
        let action = *docker_action;
        ws.handle(event, move |conn: Arc<Conn>, msg: ClientMessage| {
            let state = state.clone();
            async move {
                let uid = state.check_login(&conn, &msg).await;
                if uid == 0 {
                    return;
                }

                let args = parse_args(&msg);
                let container_name = arg_string(&args, 0);
                if container_name.is_empty() {
                    if let Some(id) = msg.id {
                        conn.send_ack(id, ErrorResponse::new("Container name required"))
                            .await;
                    }
                    return;
                }

                if let Some(id) = msg.id {
                    conn.send_ack(id, OkResponse { ok: true, msg: None, token: None }).await;
                }

                let result = match action {
                    DockerAction::Start => {
                        docker::with_timeout(state.docker.start_container(
                            &container_name,
                            None::<bollard::query_parameters::StartContainerOptions>,
                        ))
                        .await
                    }
                    DockerAction::Stop => {
                        docker::with_timeout(state.docker.stop_container(
                            &container_name,
                            None::<bollard::query_parameters::StopContainerOptions>,
                        ))
                        .await
                    }
                    DockerAction::Restart => {
                        docker::with_timeout(state.docker.restart_container(
                            &container_name,
                            None::<bollard::query_parameters::RestartContainerOptions>,
                        ))
                        .await
                    }
                };

                if let Err(e) = result {
                    warn!(container = %container_name, "container action failed: {e}");
                }
            }
        });
    }
}

#[derive(Clone, Copy)]
enum DockerAction {
    Start,
    Stop,
    Restart,
}
