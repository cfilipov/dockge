use crate::handlers::auth::check_login;
use crate::error::error_response;
use crate::models::stack::Stack;
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
        socket.on("terminalJoin", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let state = state.clone();
            tokio::spawn(async move {
                let result = handle_terminal_join(&state, &socket, &data).await;
                ack.send(&result).ok();
            });
        });
    }

    // terminalInput
    {
        let state = state.clone();
        socket.on("terminalInput", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let state = state.clone();
            tokio::spawn(async move {
                let result = handle_terminal_input(&state, &socket, &data).await;
                ack.send(&result).ok();
            });
        });
    }

    // terminalResize
    {
        let _state = state.clone();
        socket.on("terminalResize", move |_socket: SocketRef, Data(data): Data<Value>| {
            // No callback, fire and forget
            let terminal_name = if data.is_array() {
                data.get(0).and_then(|v| v.as_str()).unwrap_or("").to_string()
            } else {
                String::new()
            };
            debug!("Terminal resize requested: {}", terminal_name);
            // In a full PTY implementation, we'd resize the PTY here
        });
    }

    // leaveCombinedTerminal
    {
        let state = state.clone();
        socket.on("leaveCombinedTerminal", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let _state = state.clone();
            tokio::spawn(async move {
                if check_login(&socket).is_none() {
                    ack.send(&error_response("Not logged in")).ok();
                    return;
                }
                let stack_name = if data.is_array() {
                    data.get(0).and_then(|v| v.as_str()).unwrap_or("")
                } else {
                    data.as_str().unwrap_or("")
                };
                debug!("Left combined terminal for stack: {}", stack_name);
                ack.send(&json!({ "ok": true })).ok();
            });
        });
    }

    // interactiveTerminal
    {
        let state = state.clone();
        socket.on("interactiveTerminal", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let state = state.clone();
            let socket = socket.clone();
            tokio::spawn(async move {
                let result = handle_interactive_terminal(&state, &socket, &data).await;
                ack.send(&result).ok();
            });
        });
    }

    // joinContainerLog
    {
        let state = state.clone();
        socket.on("joinContainerLog", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let state = state.clone();
            let socket = socket.clone();
            tokio::spawn(async move {
                let result = handle_join_container_log(&state, &socket, &data).await;
                ack.send(&result).ok();
            });
        });
    }

    // mainTerminal
    {
        let state = state.clone();
        socket.on("mainTerminal", move |socket: SocketRef, Data(_data): Data<Value>, ack: AckSender| {
            let state = state.clone();
            tokio::spawn(async move {
                if !state.config.enable_console {
                    ack.send(&error_response("Console is not enabled")).ok();
                    return;
                }
                if check_login(&socket).is_none() {
                    ack.send(&error_response("Not logged in")).ok();
                    return;
                }
                ack.send(&json!({ "ok": true })).ok();
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

async fn handle_terminal_join(state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
    if check_login(socket).is_none() {
        return error_response("Not logged in");
    }

    let terminal_name = if data.is_array() {
        data.get(0).and_then(|v| v.as_str()).unwrap_or("")
    } else {
        data.as_str().unwrap_or("")
    };

    // Return the buffer contents
    let terminals = state.terminals.read().await;
    if let Some(handle) = terminals.get(terminal_name) {
        let buffer = handle.buffer.join("");
        json!({ "ok": true, "buffer": buffer })
    } else {
        json!({ "ok": true, "buffer": "" })
    }
}

async fn handle_terminal_input(_state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
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

    // In a full implementation, we'd write to the terminal's stdin
    debug!("Terminal input: {} -> {}", terminal_name, cmd);

    json!({ "ok": true })
}

async fn handle_interactive_terminal(state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
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

    // Spawn the interactive terminal in background
    let socket_clone = socket.clone();
    let terminal_name_clone = terminal_name.clone();
    let path = stack_obj.path();

    tokio::spawn(async move {
        let (tx, _) = tokio::sync::broadcast::channel(256);
        let mut rx = tx.subscribe();

        let output_task = tokio::spawn(async move {
            while let Ok((term_name, data)) = rx.recv().await {
                socket_clone.emit("agent", &("terminalWrite", &term_name, &data)).ok();
            }
        });

        let mut cmd = tokio::process::Command::new("docker");
        cmd.args(&args)
            .current_dir(&path)
            .stdin(std::process::Stdio::piped())
            .stdout(std::process::Stdio::piped())
            .stderr(std::process::Stdio::piped());

        match cmd.spawn() {
            Ok(mut child) => {
                // Stream output
                if let Some(stdout) = child.stdout.take() {
                    let term_name = terminal_name_clone.clone();
                    tokio::spawn(async move {
                        use tokio::io::{AsyncBufReadExt, BufReader};
                        let reader = BufReader::new(stdout);
                        let mut lines = reader.lines();
                        while let Ok(Some(line)) = lines.next_line().await {
                            let _ = tx.send((term_name.clone(), format!("{}\r\n", line)));
                        }
                    });
                }
                let _ = child.wait().await;
            }
            Err(e) => {
                warn!("Failed to spawn interactive terminal: {}", e);
            }
        }

        drop(output_task);
    });

    json!({ "ok": true })
}

async fn handle_join_container_log(state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
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

    // Spawn log streaming in background
    let socket_clone = socket.clone();
    let path = stack_obj.path();

    tokio::spawn(async move {
        let (tx, _) = tokio::sync::broadcast::channel(256);
        let mut rx = tx.subscribe();

        let output_task = tokio::spawn({
            let socket = socket_clone.clone();
            async move {
                while let Ok((term_name, data)) = rx.recv().await {
                    socket.emit("agent", &("terminalWrite", &term_name, &data)).ok();
                }
            }
        });

        let mut cmd = tokio::process::Command::new("docker");
        cmd.args(&args)
            .current_dir(&path)
            .stdout(std::process::Stdio::piped())
            .stderr(std::process::Stdio::piped());

        match cmd.spawn() {
            Ok(mut child) => {
                if let Some(stdout) = child.stdout.take() {
                    let term_name = terminal_name.clone();
                    tokio::spawn(async move {
                        use tokio::io::{AsyncBufReadExt, BufReader};
                        let reader = BufReader::new(stdout);
                        let mut lines = reader.lines();
                        while let Ok(Some(line)) = lines.next_line().await {
                            let _ = tx.send((term_name.clone(), format!("{}\r\n", line)));
                        }
                    });
                }
                let _ = child.wait().await;
            }
            Err(e) => {
                warn!("Failed to spawn container log: {}", e);
            }
        }

        drop(output_task);
    });

    json!({ "ok": true })
}
