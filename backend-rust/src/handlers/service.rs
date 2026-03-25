use std::sync::Arc;
use std::time::Duration;

use tracing::{info, warn};

use crate::ws::conn::Conn;
use crate::ws::protocol::{ActionCompleteEvent, ClientMessage, ErrorResponse, OkResponse};
use crate::ws::WsServer;

use super::stack::is_stack_managed;
use super::{arg_string, parse_args, run_pty_to_terminal, AppState};

/// Service action descriptor for the loop-registered handlers.
struct ServiceAction {
    event: &'static str,
    /// Compose subcommand args for managed stacks (e.g. ["up", "-d", SERVICE] or ["stop", SERVICE]).
    /// The service name placeholder is appended at call time.
    compose_args: &'static [&'static str],
    /// Plain docker action for unmanaged containers (e.g. "start", "stop", "restart").
    docker_action: &'static str,
}

const SERVICE_ACTIONS: &[ServiceAction] = &[
    ServiceAction {
        event: "startService",
        compose_args: &["up", "-d"],
        docker_action: "start",
    },
    ServiceAction {
        event: "stopService",
        compose_args: &["stop"],
        docker_action: "stop",
    },
    ServiceAction {
        event: "restartService",
        compose_args: &["restart"],
        docker_action: "restart",
    },
    ServiceAction {
        event: "recreateService",
        compose_args: &["up", "-d", "--force-recreate"],
        docker_action: "restart",
    },
    ServiceAction {
        event: "updateService",
        compose_args: &["restart"],
        docker_action: "restart",
    },
];

pub fn register(ws: &mut WsServer, state: Arc<AppState>) {
    // Per-service lifecycle: run via PTY terminal so output streams to the UI
    for sa in SERVICE_ACTIONS {
        let state = state.clone();
        let compose_args: Vec<String> = sa.compose_args.iter().map(|s| (*s).to_string()).collect();
        let docker_action = sa.docker_action.to_string();
        let is_update = sa.event == "updateService";
        let is_recreate = sa.event == "recreateService";
        ws.handle(sa.event, move |conn: Arc<Conn>, msg: ClientMessage| {
            let state = state.clone();
            let compose_args = compose_args.clone();
            let docker_action = docker_action.clone();
            async move {
                let uid = state.check_login(&conn, &msg);
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
                        );
                    }
                    return;
                }

                let managed = is_stack_managed(&state.config.stacks_dir, &stack_name);

                // recreateService and updateService require a managed stack
                if (is_recreate || is_update) && !managed {
                    let action_label = if is_recreate { "recreate" } else { "update" };
                    if let Some(id) = msg.id {
                        conn.send_ack(
                            id,
                            ErrorResponse::new(format!(
                                "Cannot {action_label}: stack is not managed by Dockge"
                            )),
                        );
                    }
                    return;
                }

                // Ack immediately
                let request_id = msg.id;
                if let Some(id) = request_id {
                    conn.send_ack(id, OkResponse::simple());
                }

                // Run in background so the worker is free for terminalJoin etc.
                tokio::spawn(async move {
                    if managed {
                        if is_update {
                            // Update: pull first, then up -d --force-recreate
                            run_service_compose_action(
                                &state,
                                &stack_name,
                                &service_name,
                                &["pull"],
                            )
                            .await;
                            run_service_compose_action(
                                &state,
                                &stack_name,
                                &service_name,
                                &["up", "-d", "--force-recreate"],
                            )
                            .await;
                        } else {
                            run_service_compose_action(
                                &state,
                                &stack_name,
                                &service_name,
                                &compose_args.iter().map(|s| s.as_str()).collect::<Vec<_>>(),
                            )
                            .await;
                        }
                    } else {
                        // Unmanaged: plain docker command on the container
                        let container_name = format!("{stack_name}-{service_name}-1");
                        run_container_action_for_stack(
                            &state,
                            &stack_name,
                            &container_name,
                            &docker_action,
                        )
                        .await;
                    }
                    conn.send_event("actionComplete", ActionCompleteEvent { request_id, ok: true, msg: None });
                });
            }
        });
    }

    // Per-container lifecycle (standalone, no compose project): run via PTY terminal
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
                let uid = state.check_login(&conn, &msg);
                if uid == 0 {
                    return;
                }

                let args = parse_args(&msg);
                let container_name = arg_string(&args, 0);
                if container_name.is_empty() {
                    if let Some(id) = msg.id {
                        conn.send_ack(id, ErrorResponse::new("Container name required"));
                    }
                    return;
                }

                // Create the terminal before ack so it exists when terminalJoin arrives
                let term_name = format!("container-{container_name}");
                state.terminal_manager.get_or_create(&term_name, true).await;

                let request_id = msg.id;
                if let Some(id) = request_id {
                    conn.send_ack(id, OkResponse::simple());
                }

                // Run action in background via PTY terminal
                let state = state.clone();
                tokio::spawn(async move {
                    run_container_action(&state, &container_name, &action).await;
                    conn.send_event("actionComplete", ActionCompleteEvent { request_id, ok: true, msg: None });
                });
            }
        });
    }
}

/// Run a compose command for a single service in a managed stack,
/// streaming output through a PTY terminal on "compose-{stackName}".
async fn run_service_compose_action(
    state: &AppState,
    stack_name: &str,
    service_name: &str,
    compose_args: &[&str],
) {
    let term_name = format!("compose-{stack_name}");
    let stack_dir = format!("{}/{}", state.config.stacks_dir, stack_name);

    let mut args: Vec<&str> = vec!["compose"];
    args.extend_from_slice(compose_args);
    args.push(service_name);

    let cmd_display = format!("$ docker {}\r\n", args.join(" "));

    state.terminal_manager.get_or_create(&term_name, true).await;
    state.terminal_manager.write_data(&term_name, cmd_display.into_bytes());

    match run_pty_to_terminal(&state.terminal_manager, &term_name, "docker", &args, Some(&stack_dir)).await {
        Ok(()) => info!(stack = %stack_name, service = %service_name, "service compose action completed"),
        Err(e) => warn!(stack = %stack_name, service = %service_name, "service compose action failed: {e}"),
    }

    state.terminal_manager.remove_after(&term_name, Duration::from_secs(30));
}

/// Run a plain docker command (start/stop/restart) for an unmanaged service container,
/// writing output to the stack's compose terminal so the UI progress terminal shows it.
async fn run_container_action_for_stack(
    state: &AppState,
    stack_name: &str,
    container_name: &str,
    action: &str,
) {
    let term_name = format!("compose-{stack_name}");
    let cmd_display = format!("$ docker {action} {container_name}\r\n");

    state.terminal_manager.get_or_create(&term_name, true).await;
    state.terminal_manager.write_data(&term_name, cmd_display.into_bytes());

    match run_pty_to_terminal(&state.terminal_manager, &term_name, "docker", &[action, container_name], None).await {
        Ok(()) => info!(stack = %stack_name, container = %container_name, action = %action, "unmanaged container action completed"),
        Err(e) => warn!(stack = %stack_name, container = %container_name, action = %action, "unmanaged container action failed: {e}"),
    }

    state.terminal_manager.remove_after(&term_name, Duration::from_secs(30));
}

/// Run a plain docker command (start/stop/restart) for a standalone container,
/// streaming output through a PTY terminal.
async fn run_container_action(state: &AppState, container_name: &str, action: &str) {
    let term_name = format!("container-{container_name}");
    let cmd_display = format!("$ docker {action} {container_name}\r\n");

    // Terminal already created by the handler before ack; just write the command display
    state.terminal_manager.write_data(&term_name, cmd_display.into_bytes());

    match run_pty_to_terminal(&state.terminal_manager, &term_name, "docker", &[action, container_name], None).await {
        Ok(()) => info!(container = %container_name, action = %action, "container action completed"),
        Err(e) => warn!(container = %container_name, action = %action, "container action failed: {e}"),
    }

    state.terminal_manager.remove_after(&term_name, Duration::from_secs(30));
}
