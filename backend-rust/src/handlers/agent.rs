use crate::docker;
use crate::error::error_response;
use crate::handlers::auth::check_login;
use crate::state::AppState;
use serde_json::{json, Value};
use socketioxide::extract::{Data, SocketRef, AckSender};
use std::sync::Arc;
use tracing::debug;

/// Register agent management socket event handlers
pub fn register(socket: &SocketRef, _state: Arc<AppState>) {
    // addAgent (stub)
    {
        socket.on("addAgent", move |_socket: SocketRef, ack: AckSender| {
            ack.send(&json!({ "ok": false, "msg": "Agent management is not yet supported in the Rust backend" })).ok();
        });
    }

    // removeAgent (stub)
    {
        socket.on("removeAgent", move |_socket: SocketRef, ack: AckSender| {
            ack.send(&json!({ "ok": false, "msg": "Agent management is not yet supported in the Rust backend" })).ok();
        });
    }

    // updateAgent (stub)
    {
        socket.on("updateAgent", move |_socket: SocketRef, ack: AckSender| {
            ack.send(&json!({ "ok": false, "msg": "Agent management is not yet supported in the Rust backend" })).ok();
        });
    }
}

/// Register the agent proxy handler.
/// The frontend sends `socket.emit("agent", endpoint, eventName, ...args, callback)`.
/// We need to route this to the correct handler.
pub fn register_agent_proxy(socket: &SocketRef, state: Arc<AppState>) {
    let state = state.clone();

    socket.on("agent", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
        let state = state.clone();
        let socket = socket.clone();

        tokio::spawn(async move {
            // data is [endpoint, eventName, ...args]
            let arr = match data.as_array() {
                Some(a) if a.len() >= 2 => a,
                _ => {
                    ack.send(&json!({ "ok": false, "msg": "Invalid agent call" })).ok();
                    return;
                }
            };

            let _endpoint = arr[0].as_str().unwrap_or("");
            let event_name = arr[1].as_str().unwrap_or("");
            let remaining_args: Value = Value::Array(arr[2..].to_vec());

            debug!("Agent proxy: event={}, args_count={}", event_name, arr.len() - 2);

            route_agent_event(&state, &socket, event_name, &remaining_args, ack).await;
        });
    });
}

/// Route an agent event to the correct handler
async fn route_agent_event(
    state: &Arc<AppState>,
    socket: &SocketRef,
    event: &str,
    args: &Value,
    ack: AckSender,
) {
    use crate::handlers::{stack, terminal};

    match event {
        // --- Stack list ---
        "requestStackList" => {
            if check_login(socket).is_none() {
                ack.send(&error_response("Not logged in")).ok();
                return;
            }
            stack::refresh_stack_cache(state).await;
            stack::broadcast_stack_list(state).await;
            ack.send(&json!({ "ok": true, "msg": "Updated", "msgi18n": true })).ok();
        }

        // --- Stack CRUD ---
        "getStack" => {
            let result = stack::handle_get_stack(state, socket, args).await;
            ack.send(&result).ok();
        }
        "deployStack" => {
            let result = stack::handle_deploy_stack(state, socket, args).await;
            ack.send(&result).ok();
        }
        "saveStack" => {
            let result = stack::handle_save_stack(state, socket, args).await;
            ack.send(&result).ok();
        }

        // --- Compose lifecycle ---
        "startStack" => {
            let result = stack::handle_compose_action(state, socket, args, "up", &["-d", "--remove-orphans"]).await;
            ack.send(&result).ok();
        }
        "stopStack" => {
            let result = stack::handle_compose_action(state, socket, args, "stop", &[]).await;
            ack.send(&result).ok();
        }
        "restartStack" => {
            let result = stack::handle_compose_action(state, socket, args, "restart", &[]).await;
            ack.send(&result).ok();
        }
        "downStack" => {
            let result = stack::handle_compose_action(state, socket, args, "down", &[]).await;
            ack.send(&result).ok();
        }
        "updateStack" => {
            let result = stack::handle_update_stack(state, socket, args).await;
            ack.send(&result).ok();
        }

        // --- Stack deletion ---
        "deleteStack" => {
            let result = stack::handle_delete_stack(state, socket, args, false).await;
            ack.send(&result).ok();
        }
        "forceDeleteStack" => {
            let result = stack::handle_delete_stack(state, socket, args, true).await;
            ack.send(&result).ok();
        }

        // --- Service operations ---
        "serviceStatusList" => {
            let result = stack::handle_service_status_list(state, socket, args).await;
            ack.send(&result).ok();
        }
        "startService" => {
            let result = stack::handle_service_action(state, socket, args, "up").await;
            ack.send(&result).ok();
        }
        "stopService" => {
            let result = stack::handle_service_action(state, socket, args, "stop").await;
            ack.send(&result).ok();
        }
        "restartService" => {
            let result = stack::handle_service_action(state, socket, args, "restart").await;
            ack.send(&result).ok();
        }
        "updateService" => {
            let result = stack::handle_update_service(state, socket, args).await;
            ack.send(&result).ok();
        }

        // --- Image updates ---
        "checkImageUpdates" => {
            let result = stack::handle_check_image_updates(state, socket, args).await;
            ack.send(&result).ok();
        }

        // --- Docker queries ---
        "getDockerNetworkList" => {
            if check_login(socket).is_none() {
                ack.send(&error_response("Not logged in")).ok();
                return;
            }
            match docker::get_network_list().await {
                Ok(list) => { ack.send(&json!({ "ok": true, "dockerNetworkList": list })).ok(); }
                Err(e) => { ack.send(&error_response(&format!("{}", e))).ok(); }
            }
        }
        "dockerStats" => {
            if check_login(socket).is_none() {
                ack.send(&error_response("Not logged in")).ok();
                return;
            }
            match docker::get_docker_stats().await {
                Ok(stats) => { ack.send(&json!({ "ok": true, "dockerStats": stats })).ok(); }
                Err(e) => { ack.send(&error_response(&format!("{}", e))).ok(); }
            }
        }
        "containerInspect" => {
            if check_login(socket).is_none() {
                ack.send(&error_response("Not logged in")).ok();
                return;
            }
            let container_name = if args.is_array() {
                args.get(0).and_then(|v| v.as_str()).unwrap_or("")
            } else {
                args.as_str().unwrap_or("")
            };
            match docker::container_inspect(container_name).await {
                Ok(inspect_data) => {
                    ack.send(&json!({ "ok": true, "inspectData": inspect_data })).ok();
                }
                Err(e) => { ack.send(&error_response(&format!("{}", e))).ok(); }
            }
        }

        // --- Terminal operations ---
        "terminalJoin" => {
            let result = terminal::handle_terminal_join(state, socket, args).await;
            ack.send(&result).ok();
        }
        "terminalInput" => {
            let result = terminal::handle_terminal_input(state, socket, args).await;
            ack.send(&result).ok();
        }
        "terminalResize" => {
            // Fire-and-forget, no meaningful response needed
            let terminal_name = if args.is_array() {
                args.get(0).and_then(|v| v.as_str()).unwrap_or("").to_string()
            } else {
                String::new()
            };
            debug!("Terminal resize requested: {}", terminal_name);
            ack.send(&json!({ "ok": true })).ok();
        }
        "leaveCombinedTerminal" => {
            if check_login(socket).is_none() {
                ack.send(&error_response("Not logged in")).ok();
                return;
            }
            let stack_name = if args.is_array() {
                args.get(0).and_then(|v| v.as_str()).unwrap_or("")
            } else {
                args.as_str().unwrap_or("")
            };
            debug!("Left combined terminal for stack: {}", stack_name);
            ack.send(&json!({ "ok": true })).ok();
        }
        "interactiveTerminal" => {
            let result = terminal::handle_interactive_terminal(state, socket, args).await;
            ack.send(&result).ok();
        }
        "joinContainerLog" => {
            let result = terminal::handle_join_container_log(state, socket, args).await;
            ack.send(&result).ok();
        }
        "mainTerminal" => {
            if !state.config.enable_console {
                ack.send(&error_response("Console is not enabled")).ok();
                return;
            }
            if check_login(socket).is_none() {
                ack.send(&error_response("Not logged in")).ok();
                return;
            }
            ack.send(&json!({ "ok": true })).ok();
        }
        "checkMainTerminal" => {
            ack.send(&json!({ "ok": state.config.enable_console })).ok();
        }

        _ => {
            debug!("Unknown agent event: {}", event);
            ack.send(&json!({ "ok": false, "msg": format!("Unknown event: {}", event) })).ok();
        }
    }
}
