use crate::handlers::auth::check_login;
use crate::error::error_response;
use crate::models::stack::Stack;
use crate::socket_args::SocketArgs;
use crate::state::AppState;
use crate::terminal::*;
use serde_json::{json, Value};
use socketioxide::extract::{Data, SocketRef, AckSender};
use std::sync::Arc;
use tracing::{debug, warn};

/// Register terminal-related socket event handlers
pub fn register_agent_handlers(socket: &SocketRef, state: Arc<AppState>) {
    // terminalJoin
    {
        let state = state.clone();
        socket.on("terminalJoin", move |socket: SocketRef, Data(args): Data<SocketArgs>, ack: AckSender| {
            let state = state.clone();
            tokio::spawn(async move {
                let data = Value::Array(args.0);
                let result = handle_terminal_join(&state, &socket, &data).await;
                ack.send(&result).ok();
            });
        });
    }

    // terminalInput
    {
        let state = state.clone();
        socket.on("terminalInput", move |socket: SocketRef, Data(args): Data<SocketArgs>, ack: AckSender| {
            let state = state.clone();
            tokio::spawn(async move {
                let data = Value::Array(args.0);
                let result = handle_terminal_input(&state, &socket, &data).await;
                ack.send(&result).ok();
            });
        });
    }

    // terminalResize
    {
        socket.on("terminalResize", move |_socket: SocketRef, Data(args): Data<SocketArgs>| {
            let data = Value::Array(args.0);
            let terminal_name = if data.is_array() {
                data.get(0).and_then(|v| v.as_str()).unwrap_or("").to_string()
            } else {
                String::new()
            };
            let rows = if data.is_array() {
                data.get(1).and_then(|v| v.as_u64()).unwrap_or(TERMINAL_ROWS as u64) as u16
            } else {
                TERMINAL_ROWS
            };
            let cols = if data.is_array() {
                data.get(2).and_then(|v| v.as_u64()).unwrap_or(TERMINAL_COLS as u64) as u16
            } else {
                TERMINAL_COLS
            };
            tokio::spawn(async move {
                TERMINAL_MANAGER.resize(&terminal_name, rows, cols).await;
            });
        });
    }

    // leaveCombinedTerminal
    {
        let state = state.clone();
        socket.on("leaveCombinedTerminal", move |socket: SocketRef, Data(args): Data<SocketArgs>, ack: AckSender| {
            let _state = state.clone();
            tokio::spawn(async move {
                let data = Value::Array(args.0);
                if check_login(&socket).is_none() {
                    ack.send(&error_response("Not logged in")).ok();
                    return;
                }
                let stack_name = if data.is_array() {
                    data.get(0).and_then(|v| v.as_str()).unwrap_or("")
                } else {
                    data.as_str().unwrap_or("")
                };
                let terminal_name = get_combined_terminal_name("", stack_name);
                TERMINAL_MANAGER.remove(&terminal_name).await;
                debug!("Left combined terminal for stack: {}", stack_name);
                ack.send(&json!({ "ok": true })).ok();
            });
        });
    }

    // interactiveTerminal
    {
        let state = state.clone();
        socket.on("interactiveTerminal", move |socket: SocketRef, Data(args): Data<SocketArgs>, ack: AckSender| {
            let state = state.clone();
            let socket = socket.clone();
            tokio::spawn(async move {
                let data = Value::Array(args.0);
                let result = handle_interactive_terminal(&state, &socket, &data).await;
                ack.send(&result).ok();
            });
        });
    }

    // joinContainerLog
    {
        let state = state.clone();
        socket.on("joinContainerLog", move |socket: SocketRef, Data(args): Data<SocketArgs>, ack: AckSender| {
            let state = state.clone();
            let socket = socket.clone();
            tokio::spawn(async move {
                let data = Value::Array(args.0);
                let result = handle_join_container_log(&state, &socket, &data).await;
                ack.send(&result).ok();
            });
        });
    }

    // mainTerminal
    {
        let state = state.clone();
        socket.on("mainTerminal", move |socket: SocketRef, Data(_args): Data<SocketArgs>, ack: AckSender| {
            let state = state.clone();
            let socket = socket.clone();
            tokio::spawn(async move {
                if !state.config.enable_console {
                    ack.send(&error_response("Console is not enabled")).ok();
                    return;
                }
                if check_login(&socket).is_none() {
                    ack.send(&error_response("Not logged in")).ok();
                    return;
                }
                let result = handle_main_terminal(&state, &socket).await;
                ack.send(&result).ok();
            });
        });
    }

    // checkMainTerminal
    {
        let state = state.clone();
        socket.on("checkMainTerminal", move |_socket: SocketRef, ack: AckSender| {
            let state = state.clone();
            tokio::spawn(async move {
                ack.send(&json!({ "ok": state.config.enable_console })).ok();
            });
        });
    }
}

pub async fn handle_terminal_join(_state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
    if check_login(socket).is_none() {
        return error_response("Not logged in");
    }

    let terminal_name = if data.is_array() {
        data.get(0).and_then(|v| v.as_str()).unwrap_or("")
    } else {
        data.as_str().unwrap_or("")
    };

    let buffer = TERMINAL_MANAGER.get_buffer(terminal_name).await;
    json!({ "ok": true, "buffer": buffer })
}

pub async fn handle_terminal_input(_state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
    if check_login(socket).is_none() {
        return error_response("Not logged in");
    }

    let arr = match data.as_array() {
        Some(a) => a,
        None => return error_response("Invalid data"),
    };

    let terminal_name = arr.first().and_then(|v| v.as_str()).unwrap_or("");
    let cmd = arr.get(1).and_then(|v| v.as_str()).unwrap_or("");

    if terminal_name.is_empty() {
        return error_response("Terminal name required");
    }

    TERMINAL_MANAGER.write_input(terminal_name, cmd.as_bytes()).await;

    json!({ "ok": true })
}

pub async fn handle_interactive_terminal(state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
    if check_login(socket).is_none() {
        return error_response("Not logged in");
    }

    let arr = match data.as_array() {
        Some(a) => a,
        None => return error_response("Invalid data"),
    };

    let stack_name = arr.first().and_then(|v| v.as_str()).unwrap_or("");
    let service_name = arr.get(1).and_then(|v| v.as_str()).unwrap_or("");
    let shell = arr.get(2).and_then(|v| v.as_str()).unwrap_or("sh");

    if stack_name.is_empty() || service_name.is_empty() {
        return error_response("Stack name and service name required");
    }

    let stack_obj = match Stack::get_stack(&state.stacks_dir, stack_name).await {
        Ok(s) => s,
        Err(e) => return error_response(&format!("{}", e)),
    };

    let terminal_name = get_container_exec_terminal_name("", stack_name, service_name, 0);
    let args = stack_obj.get_compose_options("exec", &[service_name, shell]);
    let path = stack_obj.path();

    match TERMINAL_MANAGER
        .spawn_persistent(&terminal_name, "docker", &args, &path, TERMINAL_ROWS, TERMINAL_COLS)
        .await
    {
        Ok(terminal) => {
            // Subscribe and forward output to socket
            let mut rx = terminal.subscribe();
            let terminal_for_exit = terminal.clone();
            let socket_clone = socket.clone();
            let term_name = terminal_name.clone();
            tokio::spawn(async move {
                while let Ok((name, data)) = rx.recv().await {
                    socket_clone.emit("agent", &("terminalWrite", &name, &data)).ok();
                }
                let exit_code = terminal_for_exit.wait_for_exit().await.unwrap_or(0);
                socket_clone.emit("agent", &("terminalExit", &term_name, exit_code)).ok();
            });

            json!({ "ok": true })
        }
        Err(e) => {
            warn!("Failed to spawn interactive terminal: {}", e);
            error_response(&format!("Failed to spawn terminal: {}", e))
        }
    }
}

pub async fn handle_join_container_log(state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
    if check_login(socket).is_none() {
        return error_response("Not logged in");
    }

    let arr = match data.as_array() {
        Some(a) => a,
        None => return error_response("Invalid data"),
    };

    let stack_name = arr.first().and_then(|v| v.as_str()).unwrap_or("");
    let service_name = arr.get(1).and_then(|v| v.as_str()).unwrap_or("");

    let stack_obj = match Stack::get_stack(&state.stacks_dir, stack_name).await {
        Ok(s) => s,
        Err(e) => return error_response(&format!("{}", e)),
    };

    let terminal_name = get_container_log_name("", stack_name, service_name);
    let args = stack_obj.get_compose_options("logs", &["-f", "--tail", "100", service_name]);
    let path = stack_obj.path();

    match TERMINAL_MANAGER
        .spawn_persistent(&terminal_name, "docker", &args, &path, TERMINAL_ROWS, TERMINAL_COLS)
        .await
    {
        Ok(terminal) => {
            // Send existing buffer first
            let buffer = terminal.get_buffer().await;
            if !buffer.is_empty() {
                socket.emit("agent", &("terminalWrite", &terminal_name, &buffer)).ok();
            }

            // Subscribe and forward output to socket
            let mut rx = terminal.subscribe();
            let terminal_for_exit = terminal.clone();
            let socket_clone = socket.clone();
            let term_name = terminal_name.clone();
            tokio::spawn(async move {
                while let Ok((name, data)) = rx.recv().await {
                    socket_clone.emit("agent", &("terminalWrite", &name, &data)).ok();
                }
                let exit_code = terminal_for_exit.wait_for_exit().await.unwrap_or(0);
                socket_clone.emit("agent", &("terminalExit", &term_name, exit_code)).ok();
            });

            json!({ "ok": true })
        }
        Err(e) => {
            warn!("Failed to spawn container log: {}", e);
            error_response(&format!("Failed to spawn log terminal: {}", e))
        }
    }
}

pub async fn handle_main_terminal(_state: &Arc<AppState>, socket: &SocketRef) -> Value {
    let terminal_name = "main-terminal".to_string();
    let cwd = std::env::current_dir().unwrap_or_else(|_| std::path::PathBuf::from("/"));

    match TERMINAL_MANAGER
        .spawn_persistent(&terminal_name, "bash", &[], &cwd, TERMINAL_ROWS, TERMINAL_COLS)
        .await
    {
        Ok(terminal) => {
            let buffer = terminal.get_buffer().await;
            if !buffer.is_empty() {
                socket.emit("agent", &("terminalWrite", &terminal_name, &buffer)).ok();
            }

            let mut rx = terminal.subscribe();
            let terminal_for_exit = terminal.clone();
            let socket_clone = socket.clone();
            let term_name = terminal_name.clone();
            tokio::spawn(async move {
                while let Ok((name, data)) = rx.recv().await {
                    socket_clone.emit("agent", &("terminalWrite", &name, &data)).ok();
                }
                let exit_code = terminal_for_exit.wait_for_exit().await.unwrap_or(0);
                socket_clone.emit("agent", &("terminalExit", &term_name, exit_code)).ok();
            });

            json!({ "ok": true })
        }
        Err(e) => {
            warn!("Failed to spawn main terminal: {}", e);
            error_response(&format!("Failed to spawn terminal: {}", e))
        }
    }
}
