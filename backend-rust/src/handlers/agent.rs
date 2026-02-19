use crate::docker;
use crate::error::error_response;
use crate::handlers::auth::{check_login, disconnect_other_clients, send_agent_list};
use crate::models::agent::Agent;
use crate::socket_args::SocketArgs;
use crate::state::AppState;
use crate::terminal::{get_combined_terminal_name, TERMINAL_MANAGER, TERMINAL_ROWS, TERMINAL_COLS};
use serde_json::{json, Value};
use socketioxide::extract::{Data, SocketRef, AckSender};
use std::sync::Arc;
use tracing::{debug, info};

/// Register agent management socket event handlers
pub fn register(socket: &SocketRef, state: Arc<AppState>) {
    // addAgent
    {
        let state = state.clone();
        socket.on("addAgent", move |socket: SocketRef, Data(args): Data<SocketArgs>, ack: AckSender| {
            let state = state.clone();
            tokio::spawn(async move {
                let data = Value::Array(args.0);
                let result = handle_add_agent(&state, &socket, &data).await;
                ack.send(&result).ok();
            });
        });
    }

    // removeAgent
    {
        let state = state.clone();
        socket.on("removeAgent", move |socket: SocketRef, Data(args): Data<SocketArgs>, ack: AckSender| {
            let state = state.clone();
            tokio::spawn(async move {
                let data = Value::Array(args.0);
                let result = handle_remove_agent(&state, &socket, &data).await;
                ack.send(&result).ok();
            });
        });
    }

    // updateAgent
    {
        let state = state.clone();
        socket.on("updateAgent", move |socket: SocketRef, Data(args): Data<SocketArgs>, ack: AckSender| {
            let state = state.clone();
            tokio::spawn(async move {
                let data = Value::Array(args.0);
                let result = handle_update_agent(&state, &socket, &data).await;
                ack.send(&result).ok();
            });
        });
    }
}

async fn handle_add_agent(state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
    if check_login(socket).is_none() {
        return error_response("Not logged in");
    }

    let agent_data = if data.is_array() {
        data.get(0).unwrap_or(data)
    } else {
        data
    };

    let url = agent_data.get("url").and_then(|v| v.as_str()).unwrap_or("");
    let username = agent_data.get("username").and_then(|v| v.as_str()).unwrap_or("");
    let password = agent_data.get("password").and_then(|v| v.as_str()).unwrap_or("");
    let name = agent_data.get("name").and_then(|v| v.as_str()).unwrap_or("");

    if url.is_empty() || username.is_empty() || password.is_empty() {
        return error_response("URL, username, and password are required");
    }

    // Check if agent already exists
    match Agent::find_by_url(&state.db, url).await {
        Ok(Some(_)) => return error_response("Agent with this URL already exists"),
        Err(e) => return error_response(&format!("Database error: {}", e)),
        _ => {}
    }

    match Agent::create(&state.db, url, username, password, name).await {
        Ok(_) => {
            info!("Added agent: {} ({})", name, url);

            // Refresh other clients so they see the updated agent list
            let socket_id = socket.id.to_string();
            disconnect_other_clients(state, &socket_id, None).await;
            send_agent_list(state, socket).await;

            json!({ "ok": true, "msg": "agentAddedSuccessfully", "msgi18n": true })
        }
        Err(e) => error_response(&format!("Failed to add agent: {}", e)),
    }
}

async fn handle_remove_agent(state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
    if check_login(socket).is_none() {
        return error_response("Not logged in");
    }

    let url = if data.is_array() {
        data.get(0).and_then(|v| v.as_str()).unwrap_or("")
    } else {
        data.as_str().unwrap_or("")
    };

    if url.is_empty() {
        return error_response("URL is required");
    }

    match Agent::delete(&state.db, url).await {
        Ok(true) => {
            info!("Removed agent: {}", url);

            let socket_id = socket.id.to_string();
            disconnect_other_clients(state, &socket_id, None).await;
            send_agent_list(state, socket).await;

            json!({ "ok": true, "msg": "agentRemovedSuccessfully", "msgi18n": true })
        }
        Ok(false) => error_response("Agent not found"),
        Err(e) => error_response(&format!("Failed to remove agent: {}", e)),
    }
}

async fn handle_update_agent(state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
    if check_login(socket).is_none() {
        return error_response("Not logged in");
    }

    // data is [url, updatedName]
    let (url, updated_name) = if data.is_array() {
        let arr = data.as_array().unwrap();
        let url = arr.first().and_then(|v| v.as_str()).unwrap_or("");
        let name = arr.get(1).and_then(|v| v.as_str()).unwrap_or("");
        (url, name)
    } else {
        return error_response("Invalid data format");
    };

    if url.is_empty() {
        return error_response("URL is required");
    }

    match Agent::update_name(&state.db, url, updated_name).await {
        Ok(true) => {
            info!("Updated agent name: {} -> {}", url, updated_name);

            let socket_id = socket.id.to_string();
            disconnect_other_clients(state, &socket_id, None).await;
            send_agent_list(state, socket).await;

            json!({ "ok": true, "msg": "agentUpdatedSuccessfully", "msgi18n": true })
        }
        Ok(false) => error_response("Agent not found"),
        Err(e) => error_response(&format!("Failed to update agent: {}", e)),
    }
}

/// Register the agent proxy handler.
/// The frontend sends `socket.emit("agent", endpoint, eventName, ...args, callback)`.
/// We need to route this to the correct handler.
pub fn register_agent_proxy(socket: &SocketRef, state: Arc<AppState>) {
    let state = state.clone();

    socket.on("agent", move |socket: SocketRef, Data(args): Data<SocketArgs>, ack: AckSender| {
        let state = state.clone();
        let socket = socket.clone();

        tokio::spawn(async move {
            // args.0 is [endpoint, eventName, ...rest]
            let arr = &args.0;
            if arr.len() < 2 {
                ack.send(&json!({ "ok": false, "msg": "Invalid agent call" })).ok();
                return;
            }

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
            let terminal_name = if args.is_array() {
                args.get(0).and_then(|v| v.as_str()).unwrap_or("").to_string()
            } else {
                String::new()
            };
            let rows = if args.is_array() {
                args.get(1).and_then(|v| v.as_u64()).unwrap_or(TERMINAL_ROWS as u64) as u16
            } else {
                TERMINAL_ROWS
            };
            let cols = if args.is_array() {
                args.get(2).and_then(|v| v.as_u64()).unwrap_or(TERMINAL_COLS as u64) as u16
            } else {
                TERMINAL_COLS
            };
            TERMINAL_MANAGER.resize(&terminal_name, rows, cols).await;
            debug!("Terminal resize: {} {}x{}", terminal_name, cols, rows);
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
            let terminal_name = get_combined_terminal_name("", stack_name);
            TERMINAL_MANAGER.remove(&terminal_name).await;
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
            let result = terminal::handle_main_terminal(state, socket).await;
            ack.send(&result).ok();
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
