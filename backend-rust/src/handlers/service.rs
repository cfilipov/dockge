use std::sync::Arc;
use std::time::Duration;

use tracing::{info, warn};

use crate::terminal::TerminalType;
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

                // Execute Docker action (timeout applied by DockerClient)
                let result = match action {
                    DockerAction::Start => {
                        state.docker.start_container(
                            &container_name,
                            None::<bollard::query_parameters::StartContainerOptions>,
                        )
                        .await
                    }
                    DockerAction::Stop => {
                        state.docker.stop_container(
                            &container_name,
                            None::<bollard::query_parameters::StopContainerOptions>,
                        )
                        .await
                    }
                    DockerAction::Restart => {
                        state.docker.restart_container(
                            &container_name,
                            None::<bollard::query_parameters::RestartContainerOptions>,
                        )
                        .await
                    }
                };

                if let Err(e) = result {
                    warn!(container = %container_name, "service action failed: {e}");
                }
            }
        });
    }

    // Per-container lifecycle: run via PTY terminal so output streams to the UI
    for (event, action_str) in &[
        ("startContainer", "start"),
        ("stopContainer", "stop"),
        ("restartContainer", "restart"),
    ] {
        let state = state.clone();
        let action = action_str.to_string();
        ws.handle(event, move |conn: Arc<Conn>, msg: ClientMessage| {
            let state = state.clone();
            let action = action.clone();
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

                // Run action in background via PTY terminal
                let state = state.clone();
                tokio::spawn(async move {
                    run_container_action(&state, &container_name, &action).await;
                });
            }
        });
    }
}

/// Run a plain docker command (start/stop/restart) for a standalone container,
/// streaming output through a PTY terminal (mirrors Go's runContainerAction).
async fn run_container_action(state: &AppState, container_name: &str, action: &str) {
    let term_name = format!("container-{container_name}");
    let cmd_display = format!("$ docker {action} {container_name}\r\n");

    state.terminal_manager.recreate(&term_name, TerminalType::Pty).await;
    state.terminal_manager.write_data(&term_name, cmd_display.into_bytes());

    match state.terminal_manager.start_pty_and_wait(
        &term_name, "docker", &[action, container_name], None,
    ).await {
        Ok((_cancel, done_rx)) => {
            match done_rx.await {
                Ok(Some(0)) | Ok(None) => {
                    state.terminal_manager.write_data(
                        &term_name, b"\r\n[Done]\r\n".to_vec(),
                    );
                    info!(container = %container_name, action = %action, "container action completed");
                }
                Ok(Some(code)) => {
                    let msg = format!("\r\n[Error] exit code {code}\r\n");
                    state.terminal_manager.write_data(&term_name, msg.into_bytes());
                    warn!(container = %container_name, action = %action, "container action failed: exit code {code}");
                }
                Err(_) => {
                    state.terminal_manager.write_data(
                        &term_name, b"\r\n[Error] process lost\r\n".to_vec(),
                    );
                    warn!(container = %container_name, action = %action, "container action: process lost");
                }
            }
        }
        Err(e) => {
            let msg = format!("\r\n[Error] {e}\r\n");
            state.terminal_manager.write_data(&term_name, msg.into_bytes());
            warn!(container = %container_name, action = %action, "container action failed to start: {e}");
        }
    }

    state.terminal_manager.remove_after(&term_name, Duration::from_secs(30));
}

#[derive(Clone, Copy)]
enum DockerAction {
    Start,
    Stop,
    Restart,
}
