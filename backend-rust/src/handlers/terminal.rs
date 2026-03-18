use std::sync::atomic::{AtomicU16, Ordering};
use std::sync::Arc;

use axum::extract::ws::Message;
use futures_util::StreamExt;
use serde::Deserialize;
use tokio_util::sync::CancellationToken;
use tracing::{debug, info, warn};

use crate::docker;
use crate::terminal::TerminalType;
use crate::ws::conn::Conn;
use crate::ws::protocol::{ClientMessage, ErrorResponse, OkResponse};
use crate::ws::WsServer;

use super::{arg_object, parse_args, AppState};

static NEXT_SESSION_ID: AtomicU16 = AtomicU16::new(1);

/// Allocate a session, register writer, replay buffer, return session ID.
fn alloc_join_and_replay(
    conn: &Arc<Conn>,
    term: &Arc<parking_lot::Mutex<crate::terminal::Terminal>>,
) -> u16 {
    let session_id = NEXT_SESSION_ID.fetch_add(1, Ordering::Relaxed);
    let session_bytes = session_id.to_be_bytes();

    let conn_for_writer = conn.clone();
    let sid = session_bytes;
    let writer_key = format!("{}-{}", conn.id, session_id);

    let buffer = {
        let mut t = term.lock();
        t.join_and_get_buffer(
            writer_key,
            Box::new(move |data: &[u8]| {
                let mut frame = Vec::with_capacity(2 + data.len());
                frame.extend_from_slice(&sid);
                frame.extend_from_slice(data);
                let conn = conn_for_writer.clone();
                tokio::spawn(async move {
                    conn.send(Message::Binary(frame.into())).await;
                });
            }),
        )
    };

    // Replay buffer
    if !buffer.is_empty() {
        let conn = conn.clone();
        tokio::spawn(async move {
            let mut frame = Vec::with_capacity(2 + buffer.len());
            frame.extend_from_slice(&session_bytes);
            frame.extend_from_slice(&buffer);
            conn.send(Message::Binary(frame.into())).await;
        });
    }

    session_id
}

pub fn register(ws: &mut WsServer, state: Arc<AppState>) {
    // terminalJoin
    {
        let state = state.clone();
        ws.handle(
            "terminalJoin",
            move |conn: Arc<Conn>, msg: ClientMessage| {
                let state = state.clone();
                async move {
                    let uid = state.check_login(&conn, &msg).await;
                    if uid == 0 {
                        return;
                    }

                    let args = parse_args(&msg);
                    #[derive(Deserialize)]
                    struct JoinArgs {
                        #[serde(rename = "type")]
                        terminal_type: String,
                        #[serde(default)]
                        stack: String,
                        #[serde(default)]
                        service: String,
                        #[serde(default)]
                        container: String,
                        #[serde(default)]
                        shell: String,
                    }

                    let join_args: JoinArgs = match arg_object(&args, 0) {
                        Some(a) => a,
                        None => {
                            if let Some(id) = msg.id {
                                conn.send_ack(id, ErrorResponse::new("Invalid arguments"))
                                    .await;
                            }
                            return;
                        }
                    };

                    let shell = if join_args.shell.is_empty() {
                        "bash".to_string()
                    } else {
                        join_args.shell.clone()
                    };

                    match join_args.terminal_type.as_str() {
                        "combined" => {
                            handle_combined(
                                &state,
                                &conn,
                                &msg,
                                &join_args.stack,
                            )
                            .await;
                        }
                        "container-log" => {
                            handle_container_log(
                                &state,
                                &conn,
                                &msg,
                                &join_args.stack,
                                &join_args.service,
                                &join_args.container,
                            )
                            .await;
                        }
                        "container-log-by-name" => {
                            handle_container_log_by_name(
                                &state,
                                &conn,
                                &msg,
                                &join_args.container,
                            )
                            .await;
                        }
                        "exec" => {
                            handle_exec(
                                &state,
                                &conn,
                                &msg,
                                &join_args.stack,
                                &join_args.service,
                                &shell,
                            )
                            .await;
                        }
                        "exec-by-name" => {
                            handle_exec_by_name(
                                &state,
                                &conn,
                                &msg,
                                &join_args.container,
                                &shell,
                            )
                            .await;
                        }
                        "console" => {
                            handle_console(&state, &conn, &msg, &shell).await;
                        }
                        "compose" => {
                            handle_compose_terminal(
                                &state,
                                &conn,
                                &msg,
                                &join_args.stack,
                            )
                            .await;
                        }
                        "container-action" => {
                            handle_container_action_terminal(
                                &state,
                                &conn,
                                &msg,
                                &join_args.container,
                            )
                            .await;
                        }
                        other => {
                            warn!("unsupported terminal type: {other}");
                            if let Some(id) = msg.id {
                                conn.send_ack(
                                    id,
                                    ErrorResponse::new(&format!(
                                        "Unsupported terminal type: {other}"
                                    )),
                                )
                                .await;
                            }
                        }
                    }
                }
            },
        );
    }

    // terminalLeave
    {
        let state = state.clone();
        ws.handle(
            "terminalLeave",
            move |conn: Arc<Conn>, msg: ClientMessage| {
                let state = state.clone();
                async move {
                    let uid = state.check_login(&conn, &msg).await;
                    if uid == 0 {
                        return;
                    }

                    let args = parse_args(&msg);
                    #[derive(Deserialize)]
                    #[serde(rename_all = "camelCase")]
                    struct LeaveArgs {
                        #[serde(default)]
                        session_id: u16,
                    }
                    if let Some(leave_args) = arg_object::<LeaveArgs>(&args, 0) {
                        let writer_key =
                            format!("{}-{}", conn.id, leave_args.session_id);
                        // Remove from all terminals with this writer key
                        state.terminal_manager.remove_writer_from_all(&writer_key);
                    }

                    if let Some(id) = msg.id {
                        conn.send_ack(
                            id,
                            OkResponse {
                                ok: true,
                                msg: None,
                                token: None,
                            },
                        )
                        .await;
                    }
                }
            },
        );
    }
}

/// Register the binary frame handler for terminal input/resize.
pub fn register_binary_handler(ws: &mut WsServer, state: Arc<AppState>) {
    ws.binary_handler = Some(Box::new(move |conn, session_id, payload| {
        if payload.is_empty() {
            return;
        }
        let op = payload[0];
        let data = &payload[1..];

        // Find the terminal associated with this session's writer key
        let writer_key = format!("{}-{}", conn.id, session_id);

        // We need to find which terminal has this writer
        // For simplicity, iterate all terminals
        let terminals = state.terminal_manager.terminals.lock();
        for term in terminals.values() {
            let t = term.lock();
            if t.writers.contains_key(&writer_key) {
                match op {
                    0x00 => {
                        // Input
                        t.input(data);
                    }
                    0x01 => {
                        // Resize: [rows:u16][cols:u16] big-endian
                        if data.len() >= 4 {
                            let rows = u16::from_be_bytes([data[0], data[1]]);
                            let cols = u16::from_be_bytes([data[2], data[3]]);
                            t.resize(rows, cols);
                        }
                    }
                    _ => {}
                }
                break;
            }
        }
    }));
}

// ── Terminal type handlers ──────────────────────────────────────────────────

async fn handle_combined(
    state: &AppState,
    conn: &Arc<Conn>,
    msg: &ClientMessage,
    stack: &str,
) {
    let session_id = NEXT_SESSION_ID.fetch_add(1, Ordering::Relaxed);
    let cancel = CancellationToken::new();

    // Store cancel token in terminal manager for cleanup
    let term_name = format!("combined-{}", stack);
    let term = state
        .terminal_manager
        .create(&term_name, TerminalType::Pipe);
    {
        let mut t = term.lock();
        t.cancel = Some(cancel.clone());
    }

    if let Some(id) = msg.id {
        conn.send_ack(
            id,
            serde_json::json!({
                "ok": true,
                "sessionId": session_id,
            }),
        )
        .await;
    }

    // Stream directly to the connection (like the original implementation)
    let conn = conn.clone();
    let docker = state.docker.clone();
    let stack = stack.to_string();
    tokio::spawn(async move {
        stream_combined_logs_direct(&docker, &stack, session_id, &conn, cancel).await;
    });
}

async fn handle_container_log(
    state: &AppState,
    conn: &Arc<Conn>,
    msg: &ClientMessage,
    stack: &str,
    service: &str,
    container: &str,
) {
    let term_name = format!("container-log-{}", service);
    let container_name = if !container.is_empty() {
        container.to_string()
    } else {
        format!("{}-{}-1", stack, service)
    };

    let term = state
        .terminal_manager
        .recreate(&term_name, TerminalType::Pipe);
    let cancel = CancellationToken::new();
    {
        let mut t = term.lock();
        t.cancel = Some(cancel.clone());
    }

    let docker = state.docker.clone();
    let cname = container_name.clone();
    let term_writer = term.clone();
    tokio::spawn(async move {
        stream_single_container_to_terminal(&docker, &cname, &term_writer, cancel).await;
    });

    let session_id = alloc_join_and_replay(conn, &term);

    if let Some(id) = msg.id {
        conn.send_ack(
            id,
            serde_json::json!({
                "ok": true,
                "sessionId": session_id,
            }),
        )
        .await;
    }
}

async fn handle_container_log_by_name(
    state: &AppState,
    conn: &Arc<Conn>,
    msg: &ClientMessage,
    container: &str,
) {
    let term_name = format!("container-log-by-name-{}", container);

    let term = state
        .terminal_manager
        .recreate(&term_name, TerminalType::Pipe);
    let cancel = CancellationToken::new();
    {
        let mut t = term.lock();
        t.cancel = Some(cancel.clone());
    }

    let docker = state.docker.clone();
    let cname = container.to_string();
    let term_writer = term.clone();
    tokio::spawn(async move {
        stream_single_container_to_terminal(&docker, &cname, &term_writer, cancel).await;
    });

    let session_id = alloc_join_and_replay(conn, &term);

    if let Some(id) = msg.id {
        conn.send_ack(
            id,
            serde_json::json!({
                "ok": true,
                "sessionId": session_id,
            }),
        )
        .await;
    }
}

async fn handle_exec(
    state: &AppState,
    conn: &Arc<Conn>,
    msg: &ClientMessage,
    stack: &str,
    service: &str,
    shell: &str,
) {
    let term_name = format!("container-exec-{}-{}-0", stack, service);

    // Check if already running
    if let Some(existing) = state.terminal_manager.get(&term_name) {
        let is_running = !existing.lock().is_closed();
        if is_running {
            let session_id = alloc_join_and_replay(conn, &existing);
            if let Some(id) = msg.id {
                conn.send_ack(
                    id,
                    serde_json::json!({"ok": true, "sessionId": session_id}),
                )
                .await;
            }
            return;
        }
    }

    let term = state
        .terminal_manager
        .recreate(&term_name, TerminalType::Pty);
    let stacks_dir = &state.config.stacks_dir;
    let stack_dir = format!("{}/{}", stacks_dir, stack);

    match state.terminal_manager.start_pty(
        &term,
        "docker",
        &["compose", "exec", service, shell],
        Some(&stack_dir),
    ) {
        Ok(_cancel) => {
            let session_id = alloc_join_and_replay(conn, &term);
            if let Some(id) = msg.id {
                conn.send_ack(
                    id,
                    serde_json::json!({"ok": true, "sessionId": session_id}),
                )
                .await;
            }
            info!(stack = %stack, service = %service, "exec terminal started");
        }
        Err(e) => {
            warn!("failed to start exec terminal: {e}");
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::new(&format!("Failed to start exec: {e}")))
                    .await;
            }
        }
    }
}

async fn handle_exec_by_name(
    state: &AppState,
    conn: &Arc<Conn>,
    msg: &ClientMessage,
    container: &str,
    shell: &str,
) {
    let term_name = format!("container-exec-by-name-{}", container);

    // Check if already running
    if let Some(existing) = state.terminal_manager.get(&term_name) {
        let is_running = !existing.lock().is_closed();
        if is_running {
            let session_id = alloc_join_and_replay(conn, &existing);
            if let Some(id) = msg.id {
                conn.send_ack(
                    id,
                    serde_json::json!({"ok": true, "sessionId": session_id}),
                )
                .await;
            }
            return;
        }
    }

    let term = state
        .terminal_manager
        .recreate(&term_name, TerminalType::Pty);

    match state.terminal_manager.start_pty(
        &term,
        "docker",
        &["exec", "-it", container, shell],
        None,
    ) {
        Ok(_cancel) => {
            let session_id = alloc_join_and_replay(conn, &term);
            if let Some(id) = msg.id {
                conn.send_ack(
                    id,
                    serde_json::json!({"ok": true, "sessionId": session_id}),
                )
                .await;
            }
            info!(container = %container, "exec-by-name terminal started");
        }
        Err(e) => {
            warn!("failed to start exec-by-name terminal: {e}");
            if let Some(id) = msg.id {
                conn.send_ack(
                    id,
                    ErrorResponse::new(&format!("Failed to start exec: {e}")),
                )
                .await;
            }
        }
    }
}

async fn handle_console(
    state: &AppState,
    conn: &Arc<Conn>,
    msg: &ClientMessage,
    shell: &str,
) {
    let term_name = "console";

    // Check if already running
    if let Some(existing) = state.terminal_manager.get(term_name) {
        let is_running = !existing.lock().is_closed();
        if is_running {
            let session_id = alloc_join_and_replay(conn, &existing);
            if let Some(id) = msg.id {
                conn.send_ack(
                    id,
                    serde_json::json!({"ok": true, "sessionId": session_id}),
                )
                .await;
            }
            return;
        }
    }

    let term = state
        .terminal_manager
        .recreate(term_name, TerminalType::Pty);

    // Detect available shell
    let shell_cmd = if shell != "bash" {
        shell.to_string()
    } else {
        // Check if bash exists
        match tokio::process::Command::new("which")
            .arg("bash")
            .output()
            .await
        {
            Ok(output) if output.status.success() => "bash".to_string(),
            _ => "sh".to_string(),
        }
    };

    let stacks_dir = state.config.stacks_dir.clone();
    match state
        .terminal_manager
        .start_pty(&term, &shell_cmd, &[], Some(&stacks_dir))
    {
        Ok(_cancel) => {
            let session_id = alloc_join_and_replay(conn, &term);
            if let Some(id) = msg.id {
                conn.send_ack(
                    id,
                    serde_json::json!({"ok": true, "sessionId": session_id}),
                )
                .await;
            }
            info!("console terminal started");
        }
        Err(e) => {
            warn!("failed to start console terminal: {e}");
            if let Some(id) = msg.id {
                conn.send_ack(
                    id,
                    ErrorResponse::new(&format!("Failed to start console: {e}")),
                )
                .await;
            }
        }
    }
}

async fn handle_compose_terminal(
    state: &AppState,
    conn: &Arc<Conn>,
    msg: &ClientMessage,
    stack: &str,
) {
    let term_name = format!("compose-{}", stack);
    let term = state.terminal_manager.get_or_create(&term_name);
    let session_id = alloc_join_and_replay(conn, &term);

    if let Some(id) = msg.id {
        conn.send_ack(
            id,
            serde_json::json!({"ok": true, "sessionId": session_id}),
        )
        .await;
    }
}

async fn handle_container_action_terminal(
    state: &AppState,
    conn: &Arc<Conn>,
    msg: &ClientMessage,
    container: &str,
) {
    let term_name = format!("container-{}", container);
    let term = state.terminal_manager.get_or_create(&term_name);
    let session_id = alloc_join_and_replay(conn, &term);

    if let Some(id) = msg.id {
        conn.send_ack(
            id,
            serde_json::json!({"ok": true, "sessionId": session_id}),
        )
        .await;
    }
}

// ── Log streaming helpers ──────────────────────────────────────────────────

/// Stream combined logs directly to a WebSocket connection as binary frames.
async fn stream_combined_logs_direct(
    docker: &bollard::Docker,
    stack: &str,
    session_id: u16,
    conn: &Conn,
    cancel: CancellationToken,
) {
    let containers = docker::container_list(docker, Some(stack))
        .await
        .unwrap_or_default();

    if containers.is_empty() {
        warn!(stack = %stack, "no containers found for combined logs");
        return;
    }

    let session_bytes = session_id.to_be_bytes();

    for container in &containers {
        if cancel.is_cancelled() {
            return;
        }

        let opts = docker::ContainerLogsOpts {
            follow: false,
            stdout: true,
            stderr: true,
            tail: "100".to_string(),
            ..Default::default()
        };

        let mut stream = std::pin::pin!(docker::container_logs(docker, &container.name, opts));

        loop {
            tokio::select! {
                () = cancel.cancelled() => return,
                item = stream.next() => {
                    match item {
                        Some(Ok(output)) => {
                            let data = output.into_bytes();
                            if data.is_empty() {
                                continue;
                            }
                            let mut frame = Vec::with_capacity(2 + data.len());
                            frame.extend_from_slice(&session_bytes);
                            frame.extend_from_slice(&data);
                            if !conn.send(Message::Binary(frame.into())).await {
                                return;
                            }
                        }
                        Some(Err(e)) => {
                            debug!(container = %container.name, "log stream error: {e}");
                            break;
                        }
                        None => break,
                    }
                }
            }
        }
    }
}

/// Stream combined logs from all containers in a stack into a terminal buffer.
async fn stream_combined_to_terminal(
    docker: &bollard::Docker,
    stack: &str,
    term: &Arc<parking_lot::Mutex<crate::terminal::Terminal>>,
    cancel: CancellationToken,
) {
    let containers = docker::container_list(docker, Some(stack))
        .await
        .unwrap_or_default();

    if containers.is_empty() {
        warn!(stack = %stack, "no containers found for combined logs");
        return;
    }

    // Stream historical logs from each container
    for container in &containers {
        if cancel.is_cancelled() {
            return;
        }

        let opts = docker::ContainerLogsOpts {
            follow: false,
            stdout: true,
            stderr: true,
            tail: "100".to_string(),
            ..Default::default()
        };

        let mut stream = std::pin::pin!(docker::container_logs(docker, &container.name, opts));

        loop {
            tokio::select! {
                () = cancel.cancelled() => return,
                item = stream.next() => {
                    match item {
                        Some(Ok(output)) => {
                            let data = output.into_bytes();
                            if !data.is_empty() {
                                term.lock().write_data(&data);
                            }
                        }
                        Some(Err(e)) => {
                            debug!(container = %container.name, "log stream error: {e}");
                            break;
                        }
                        None => break,
                    }
                }
            }
        }
    }
}

/// Stream logs from a single container into a terminal buffer.
async fn stream_single_container_to_terminal(
    docker: &bollard::Docker,
    container_name: &str,
    term: &Arc<parking_lot::Mutex<crate::terminal::Terminal>>,
    cancel: CancellationToken,
) {
    let opts = docker::ContainerLogsOpts {
        follow: true,
        stdout: true,
        stderr: true,
        tail: "100".to_string(),
        ..Default::default()
    };

    let mut stream = std::pin::pin!(docker::container_logs(docker, container_name, opts));

    loop {
        tokio::select! {
            () = cancel.cancelled() => return,
            item = stream.next() => {
                match item {
                    Some(Ok(output)) => {
                        let data = output.into_bytes();
                        if !data.is_empty() {
                            term.lock().write_data(&data);
                        }
                    }
                    Some(Err(e)) => {
                        debug!(container = %container_name, "log stream error: {e}");
                        break;
                    }
                    None => break,
                }
            }
        }
    }
}
