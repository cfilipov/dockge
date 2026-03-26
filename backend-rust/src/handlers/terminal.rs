use std::sync::atomic::{AtomicU16, Ordering};
use std::sync::Arc;

use axum::extract::ws::Message;
use futures_util::StreamExt;
use serde::Deserialize;
use tokio::sync::broadcast::error::RecvError;
use tokio::task::JoinSet;
use tokio_util::sync::CancellationToken;
use tracing::{debug, info, warn};

use crate::broadcast::eventbus::EventBus;
use crate::docker;
use crate::terminal::TerminalHandle;
use crate::ws::conn::Conn;
use crate::ws::protocol::{ClientMessage, ErrorResponse, OkResponse, SessionResponse};

/// Push event payload sent when a PTY process exits.
#[derive(serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct TerminalExitedEvent {
    session_id: u16,
}
use crate::ws::WsServer;

use super::{arg_object, parse_args, AppState};

static NEXT_SESSION_ID: AtomicU16 = AtomicU16::new(1);

// ── ANSI color helpers ──────────────────────────────────────────────────────

/// 6 rotating ANSI colors for service prefixes.
const SERVICE_COLORS: &[&str] = &[
    "\x1b[36m", // cyan
    "\x1b[33m", // yellow
    "\x1b[32m", // green
    "\x1b[35m", // magenta
    "\x1b[34m", // blue
    "\x1b[31m", // red
];
const ANSI_RESET: &str = "\x1b[0m";
fn colored_prefix(service: &str, max_len: usize, color_idx: usize) -> String {
    let color = SERVICE_COLORS[color_idx % SERVICE_COLORS.len()];
    let padded = format!("{:width$}", service, width = max_len);
    format!("{color}{padded} | {ANSI_RESET}")
}

/// Split a Docker log line with timestamp prefix into (timestamp, rest).
/// Docker timestamps are RFC3339Nano like "2024-01-15T10:30:00.123456789Z".
/// Returns (ts_str, remaining_line). If no timestamp found, returns ("", full_line).
fn split_timestamp(raw: &str) -> (&str, &str) {
    // Docker timestamps end with 'Z' or offset, followed by a space.
    // Look for the first space within 35 chars (RFC3339Nano is ~30 chars).
    if let Some(pos) = raw[..raw.len().min(35)].find(' ') {
        let ts = &raw[..pos];
        // Sanity check: timestamps start with a digit
        if ts.starts_with(|c: char| c.is_ascii_digit()) {
            return (ts, &raw[pos + 1..]);
        }
    }
    ("", raw)
}

// ── Allocate, join, replay ──────────────────────────────────────────────────

/// Allocate a session, register writer, return session ID and buffered data.
/// The caller must send the ack FIRST, then call `replay_buffer` so the
/// frontend has the session ID registered before binary frames arrive.
async fn alloc_join(
    conn: &Arc<Conn>,
    handle: &TerminalHandle,
    term_name: &str,
) -> (u16, Vec<u8>) {
    let session_id = NEXT_SESSION_ID.fetch_add(1, Ordering::Relaxed);
    let session_bytes = session_id.to_be_bytes();

    let conn_for_writer = conn.clone();
    let sid = session_bytes;
    let writer_key = format!("{}-{}", conn.id, session_id);

    let buffer = handle
        .join_and_get_buffer(
            term_name,
            writer_key,
            Box::new(move |data: &[u8]| {
                let mut frame = Vec::with_capacity(2 + data.len());
                frame.extend_from_slice(&sid);
                frame.extend_from_slice(data);
                conn_for_writer.send_nowait(Message::Binary(frame.into()));
            }),
        )
        .await;

    (session_id, buffer)
}

/// Send buffered terminal data to the client. Must be called AFTER the ack
/// so the frontend has the session ID registered to route binary frames.
fn replay_buffer(conn: &Conn, session_id: u16, buffer: Vec<u8>) {
    if buffer.is_empty() {
        return;
    }
    let session_bytes = session_id.to_be_bytes();
    let mut frame = Vec::with_capacity(2 + buffer.len());
    frame.extend_from_slice(&session_bytes);
    frame.extend_from_slice(&buffer);
    conn.send(Message::Binary(frame.into()));
}

/// Convenience: alloc+join, send ack, replay buffer (in correct order).
/// The ack must be sent before the binary replay so the frontend has the
/// session ID registered to route incoming binary frames.
async fn alloc_join_ack_replay(
    conn: &Arc<Conn>,
    handle: &TerminalHandle,
    term_name: &str,
    msg: &ClientMessage,
) -> u16 {
    let (session_id, buffer) = alloc_join(conn, handle, term_name).await;
    if let Some(id) = msg.id {
        conn.send_ack(id, SessionResponse { ok: true, session_id });
    }
    replay_buffer(conn, session_id, buffer);
    session_id
}

// ── Handler registration ────────────────────────────────────────────────────

pub fn register(ws: &mut WsServer, state: Arc<AppState>) {
    // terminalJoin
    ws.handle_with_state("terminalJoin", state.clone(), |state, conn, msg| async move {
        let uid = state.check_login(&conn, &msg);
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
                    conn.send_ack(id, ErrorResponse::new("Invalid arguments"));
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
                if join_args.stack.is_empty() {
                    send_join_error(&conn, &msg, "stack parameter required");
                    return;
                }
                handle_combined(
                    &state,
                    &conn,
                    &msg,
                    &join_args.stack,
                )
                .await;
            }
            "container-log" => {
                if join_args.stack.is_empty() || join_args.service.is_empty() {
                    send_join_error(&conn, &msg, "stack and service parameters required");
                    return;
                }
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
                if join_args.container.is_empty() {
                    send_join_error(&conn, &msg, "container parameter required");
                    return;
                }
                handle_container_log_by_name(
                    &state,
                    &conn,
                    &msg,
                    &join_args.container,
                )
                .await;
            }
            "exec" => {
                if join_args.stack.is_empty() || join_args.service.is_empty() {
                    send_join_error(&conn, &msg, "stack and service parameters required");
                    return;
                }
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
                if join_args.container.is_empty() {
                    send_join_error(&conn, &msg, "container parameter required");
                    return;
                }
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
                if join_args.stack.is_empty() {
                    send_join_error(&conn, &msg, "stack parameter required");
                    return;
                }
                handle_compose_terminal(
                    &state,
                    &conn,
                    &msg,
                    &join_args.stack,
                )
                .await;
            }
            "container-action" => {
                if join_args.container.is_empty() {
                    send_join_error(&conn, &msg, "container parameter required");
                    return;
                }
                handle_container_action_terminal(
                    &state,
                    &conn,
                    &msg,
                    &join_args.container,
                )
                .await;
            }
            other => {
                warn!("unknown terminal type: {other}");
                if let Some(id) = msg.id {
                    conn.send_ack(
                        id,
                        ErrorResponse::new(format!(
                            "unknown terminal type: {other}"
                        )),
                    );
                }
            }
        }
    });

    // clientWarning — frontend reports late log delivery or other anomalies
    ws.handle("clientWarning", move |_conn: Arc<Conn>, msg: ClientMessage| {
        async move {
            let args = parse_args(&msg);
            let warning = super::arg_string(&args, 0);
            if !warning.is_empty() {
                warn!("client warning: {warning}");
            }
        }
    });

    // terminalLeave
    ws.handle_with_state("terminalLeave", state.clone(), |state, conn, msg| async move {
        let uid = state.check_login(&conn, &msg);
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
        let leave_args = match arg_object::<LeaveArgs>(&args, 0) {
            Some(a) => a,
            None => {
                if let Some(id) = msg.id {
                    conn.send_ack(id, ErrorResponse::new("invalid args"));
                }
                return;
            }
        };

        let writer_key =
            format!("{}-{}", conn.id, leave_args.session_id);
        // Remove from all terminals with this writer key
        state.terminal_manager.remove_writer_from_all(&writer_key);

        if let Some(id) = msg.id {
            conn.send_ack(
                id,
                OkResponse::simple(),
            );
        }
    });
}

/// Send a terminalJoin error ack.
fn send_join_error(conn: &Conn, msg: &ClientMessage, err_msg: &str) {
    if let Some(id) = msg.id {
        conn.send_ack(id, ErrorResponse::new(err_msg));
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

        let writer_key = format!("{}-{}", conn.id, session_id);

        match op {
            0x00 => {
                // Input
                state
                    .terminal_manager
                    .input_by_writer_key(&writer_key, data.to_vec());
            }
            0x01 => {
                // Resize: [rows:u16][cols:u16] big-endian
                if data.len() >= 4 {
                    let rows = u16::from_be_bytes([data[0], data[1]]);
                    let cols = u16::from_be_bytes([data[2], data[3]]);
                    state
                        .terminal_manager
                        .resize_by_writer_key(&writer_key, rows, cols);
                }
            }
            _ => {}
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
    let term_name = format!("combined-{}", stack);

    // Reuse existing terminal if still active
    if let Some(false) = state.terminal_manager.is_closed(&term_name).await {
        alloc_join_ack_replay(conn, &state.terminal_manager, &term_name, msg).await;
        return;
    }

    state.terminal_manager.get_or_create(&term_name, false).await;
    let cancel = CancellationToken::new();
    state
        .terminal_manager
        .set_cancel(&term_name, cancel.clone());

    // Spawn the combined log task FIRST so buffer starts filling
    let docker = state.docker.clone();
    let handle = state.terminal_manager.clone();
    let event_bus = state.event_bus.clone();
    let stack = stack.to_string();
    let tname = term_name.clone();
    tokio::spawn(async move {
        run_combined_logs(&docker, &stack, &handle, &tname, &event_bus, cancel).await;
    });

    // THEN join — client gets any buffered data
    alloc_join_ack_replay(conn, &state.terminal_manager, &term_name, msg).await;
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

    // Reuse existing terminal if still active
    if let Some(false) = state.terminal_manager.is_closed(&term_name).await {
        alloc_join_ack_replay(conn, &state.terminal_manager, &term_name, msg).await;
        return;
    }

    state.terminal_manager.get_or_create(&term_name, false).await;
    let cancel = CancellationToken::new();
    state
        .terminal_manager
        .set_cancel(&term_name, cancel.clone());

    {
        let docker = state.docker.clone();
        let handle = state.terminal_manager.clone();
        let event_bus = state.event_bus.clone();
        let tname = term_name.clone();
        let stack_owned = stack.to_string();
        let service_owned = service.to_string();
        tokio::spawn(async move {
            let ctx = ContainerLogCtx {
                docker: &docker,
                stack: &stack_owned,
                service: &service_owned,
                handle: &handle,
                term_name: &tname,
                event_bus: &event_bus,
            };
            run_container_log_loop(&ctx, &container_name, cancel).await;
        });
    }

    alloc_join_ack_replay(conn, &state.terminal_manager, &term_name, msg).await;
}

async fn handle_container_log_by_name(
    state: &AppState,
    conn: &Arc<Conn>,
    msg: &ClientMessage,
    container: &str,
) {
    let term_name = format!("container-log-by-name-{}", container);

    // Reuse existing terminal if still active
    if let Some(false) = state.terminal_manager.is_closed(&term_name).await {
        alloc_join_ack_replay(conn, &state.terminal_manager, &term_name, msg).await;
        return;
    }

    state.terminal_manager.get_or_create(&term_name, false).await;
    let cancel = CancellationToken::new();
    state
        .terminal_manager
        .set_cancel(&term_name, cancel.clone());

    let docker = state.docker.clone();
    let handle = state.terminal_manager.clone();
    let event_bus = state.event_bus.clone();
    let cname = container.to_string();
    let tname = term_name.clone();
    tokio::spawn(async move {
        run_container_log_by_name_loop(
            &docker,
            &cname,
            &handle,
            &tname,
            &event_bus,
            cancel,
        )
        .await;
    });

    alloc_join_ack_replay(conn, &state.terminal_manager, &term_name, msg).await;
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

    // Reuse existing terminal if still active
    if let Some(false) = state.terminal_manager.is_closed(&term_name).await {
        alloc_join_ack_replay(conn, &state.terminal_manager, &term_name, msg).await;
        return;
    }

    state.terminal_manager.get_or_create(&term_name, false).await;
    let stacks_dir = &state.config.stacks_dir;
    let stack_dir = format!("{}/{}", stacks_dir, stack);

    match state.terminal_manager.start_pty_and_wait(
        &term_name,
        "docker",
        &["compose", "exec", service, shell],
        Some(&stack_dir),
    ).await {
        Ok((_cancel, done_rx)) => {
            let session_id = alloc_join_ack_replay(conn, &state.terminal_manager, &term_name, msg).await;
            info!(stack = %stack, service = %service, "exec terminal started");

            // Notify client when PTY process exits
            let conn = conn.clone();
            tokio::spawn(async move {
                let _ = done_rx.await;
                conn.send_event("terminalExited", TerminalExitedEvent { session_id });
            });
        }
        Err(e) => {
            warn!("failed to start exec terminal: {e}");
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::new(format!("Failed to start exec: {e}")));
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

    // Reuse existing terminal if still active
    if let Some(false) = state.terminal_manager.is_closed(&term_name).await {
        alloc_join_ack_replay(conn, &state.terminal_manager, &term_name, msg).await;
        return;
    }

    state.terminal_manager.get_or_create(&term_name, false).await;

    match state.terminal_manager.start_pty_and_wait(
        &term_name,
        "docker",
        &["exec", "-it", container, shell],
        None,
    ).await {
        Ok((_cancel, done_rx)) => {
            let session_id = alloc_join_ack_replay(conn, &state.terminal_manager, &term_name, msg).await;
            info!(container = %container, "exec-by-name terminal started");

            // Notify client when PTY process exits
            let conn = conn.clone();
            tokio::spawn(async move {
                let _ = done_rx.await;
                conn.send_event("terminalExited", TerminalExitedEvent { session_id });
            });
        }
        Err(e) => {
            warn!("failed to start exec-by-name terminal: {e}");
            if let Some(id) = msg.id {
                conn.send_ack(
                    id,
                    ErrorResponse::new(format!("Failed to start exec: {e}")),
                );
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

    // Reuse existing terminal if still active
    if let Some(false) = state.terminal_manager.is_closed(term_name).await {
        alloc_join_ack_replay(conn, &state.terminal_manager, term_name, msg).await;
        return;
    }

    state.terminal_manager.get_or_create(term_name, false).await;

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
        .start_pty_and_wait(term_name, &shell_cmd, &[], Some(&stacks_dir))
        .await
    {
        Ok((_cancel, done_rx)) => {
            let session_id = alloc_join_ack_replay(conn, &state.terminal_manager, term_name, msg).await;
            info!("console terminal started");

            // Notify client when PTY process exits
            let conn = conn.clone();
            tokio::spawn(async move {
                let _ = done_rx.await;
                conn.send_event("terminalExited", TerminalExitedEvent { session_id });
            });
        }
        Err(e) => {
            warn!("failed to start console terminal: {e}");
            if let Some(id) = msg.id {
                conn.send_ack(
                    id,
                    ErrorResponse::new(format!("Failed to start console: {e}")),
                );
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
    state.terminal_manager.get_or_create(&term_name, false).await;
    alloc_join_ack_replay(conn, &state.terminal_manager, &term_name, msg).await;
}

async fn handle_container_action_terminal(
    state: &AppState,
    conn: &Arc<Conn>,
    msg: &ClientMessage,
    container: &str,
) {
    let term_name = format!("container-{}", container);
    state.terminal_manager.get_or_create(&term_name, false).await;
    alloc_join_ack_replay(conn, &state.terminal_manager, &term_name, msg).await;
}

// ── Binary log entry framing ────────────────────────────────────────────────

/// Write a binary log entry: [timestamp_nanos: i64 BE, 8][message_length: u32 BE, 4][message bytes].
/// The frontend parses this structured format to get timestamps without text parsing.
fn write_log_entry(handle: &TerminalHandle, term_name: &str, nanos: i64, message: &[u8]) {
    let mut entry = Vec::with_capacity(12 + message.len());
    entry.extend_from_slice(&nanos.to_be_bytes());
    entry.extend_from_slice(&(message.len() as u32).to_be_bytes());
    entry.extend_from_slice(message);
    handle.write_data(term_name, entry);
}

// ── Combined log streaming (Phase 2) ───────────────────────────────────────

/// A log line with its original timestamp for sorting.
struct TsLine {
    /// RFC3339Nano timestamp string (sorts lexicographically for UTC —
    /// Docker always uses UTC timestamps so this is valid).
    ts: String,
    /// Nanoseconds since epoch (for binary framing).
    nanos: i64,
    /// Formatted display line with colored service prefix.
    display: String,
}

/// Two-phase combined log streamer: historical merge-sort then live follow.
async fn run_combined_logs(
    docker: &docker::DockerClient,
    stack: &str,
    handle: &TerminalHandle,
    term_name: &str,
    event_bus: &EventBus,
    cancel: CancellationToken,
) {
    // Signal readiness — triggers the frontend's firstMessage handler which
    // clears the "Connecting..." spinner. The cursor-show sequence is invisible.
    handle.write_data(term_name, b"\x1b[?25h".to_vec());

    let containers = docker::container_list(docker, Some(stack))
        .await
        .unwrap_or_default();

    // Build service → color_index mapping and compute max service name length
    // (may be empty if stack hasn't started yet — Phase 2 will pick up new containers)
    let mut max_len = containers
        .iter()
        .map(|c| c.service_name.len())
        .max()
        .unwrap_or(0);

    let mut color_map: std::collections::HashMap<String, usize> = std::collections::HashMap::new();
    for (i, c) in containers.iter().enumerate() {
        color_map.entry(c.service_name.clone()).or_insert(i);
    }

    // ── Phase 1: Historical logs with merge-sort ────────────────────────
    // Skip when no containers exist yet (e.g. newly created stack before deploy).
    if !containers.is_empty() {
        let mut all_lines: Vec<TsLine> = Vec::new();

        for container in &containers {
            if cancel.is_cancelled() {
                return;
            }

            let ci = *color_map.get(&container.service_name).unwrap_or(&0);
            let prefix = colored_prefix(&container.service_name, max_len, ci);

            let opts = docker::ContainerLogsOpts {
                follow: false,
                stdout: true,
                stderr: true,
                timestamps: true,
                tail: "100".to_string(),
                ..Default::default()
            };

            let mut stream =
                std::pin::pin!(docker::container_logs(docker, &container.name, opts));

            while let Some(item) = stream.next().await {
                if cancel.is_cancelled() {
                    return;
                }
                match item {
                    Ok(output) => {
                        let raw = output.into_bytes();
                        let text = String::from_utf8_lossy(&raw);
                        for line in text.lines() {
                            let (ts, content) = split_timestamp(line);
                            let nanos = parse_timestamp_nanos(ts).unwrap_or(0);
                            all_lines.push(TsLine {
                                ts: ts.to_string(),
                                nanos,
                                display: format!("{prefix}{content}\n"),
                            });
                        }
                    }
                    Err(e) => {
                        debug!(container = %container.name, "historical log error: {e}");
                        break;
                    }
                }
            }
        }

        // Sort by timestamp — RFC3339Nano with UTC sorts lexicographically
        all_lines.sort_by(|a, b| a.ts.cmp(&b.ts));

        // Write sorted lines to terminal buffer as binary-framed entries
        for line in &all_lines {
            if cancel.is_cancelled() {
                return;
            }
            write_log_entry(handle, term_name, line.nanos, line.display.as_bytes());
        }
    }

    // ── Phase 2: Live follow with EventBus-driven reconnection ──────────
    {
        let (line_tx, mut line_rx) = tokio::sync::mpsc::channel::<FollowLine>(256);
        let mut tasks = JoinSet::new();

        // Spawn a follower for each running container
        for container in &containers {
            let ci = *color_map.get(&container.service_name).unwrap_or(&0);
            let prefix = colored_prefix(&container.service_name, max_len, ci);
            let docker = docker.clone();
            let cname = container.name.clone();
            let tx = line_tx.clone();
            let cancel = cancel.clone();

            tasks.spawn(async move {
                follow_container_logs(&docker, &cname, &prefix, &tx, &cancel, None).await;
            });
        }

        // Subscribe to EventBus for container start/die events
        let mut event_rx = event_bus.subscribe();
        let stack_owned = stack.to_string();

        // Flush loop: read lines from followers and event bus
        loop {
            tokio::select! {
                () = cancel.cancelled() => return,
                line = line_rx.recv() => {
                    match line {
                        Some(fl) => {
                            write_log_entry(handle, term_name, fl.nanos, fl.display.as_bytes());
                        }
                        None => break, // All senders dropped
                    }
                }
                event = event_rx.recv() => {
                    match event {
                        Ok(evt) if evt.project == stack_owned && evt.event_type == "container" => {
                            if evt.action == "start" {
                                // Register new service if not yet in color_map
                                let ci = if let Some(&idx) = color_map.get(&evt.service) {
                                    idx
                                } else {
                                    let idx = color_map.len();
                                    color_map.insert(evt.service.clone(), idx);
                                    if evt.service.len() > max_len {
                                        max_len = evt.service.len();
                                    }
                                    idx
                                };
                                let prefix = colored_prefix(&evt.service, max_len, ci);

                                let docker = docker.clone();
                                let cname = evt.name.clone();
                                let tx = line_tx.clone();
                                let cancel = cancel.clone();
                                tasks.spawn(async move {
                                    follow_container_logs(
                                        &docker, &cname, &prefix, &tx, &cancel, None,
                                    )
                                    .await;
                                });
                            }
                        }
                        Err(RecvError::Lagged(_)) => {
                            // Missed events — acceptable, we'll catch up
                        }
                        Err(RecvError::Closed) => break,
                        _ => {}
                    }
                }
                // Reap completed tasks
                Some(_) = tasks.join_next() => {}
            }
        }
    }
}

/// A line from the live follow phase, with timestamp and display text.
struct FollowLine {
    nanos: i64,
    display: String,
}

/// Follow a single container's logs (live), sending prefixed lines to `tx`.
async fn follow_container_logs(
    docker: &docker::DockerClient,
    container_name: &str,
    prefix: &str,
    tx: &tokio::sync::mpsc::Sender<FollowLine>,
    cancel: &CancellationToken,
    since: Option<i32>,
) {
    let opts = docker::ContainerLogsOpts {
        follow: true,
        stdout: true,
        stderr: true,
        timestamps: true,
        tail: if since.is_some() { String::new() } else { "0".to_string() },
        since,
    };

    let mut stream = std::pin::pin!(docker::container_logs(docker, container_name, opts));

    loop {
        tokio::select! {
            () = cancel.cancelled() => return,
            item = stream.next() => {
                match item {
                    Some(Ok(output)) => {
                        let raw = output.into_bytes();
                        let text = String::from_utf8_lossy(&raw);
                        for line in text.lines() {
                            let (ts, content) = split_timestamp(line);
                            let nanos = parse_timestamp_nanos(ts).unwrap_or(0);
                            let display = format!("{prefix}{content}\n");
                            if tx.send(FollowLine { nanos, display }).await.is_err() {
                                return;
                            }
                        }
                    }
                    Some(Err(e)) => {
                        debug!(container = %container_name, "follow log error: {e}");
                        break;
                    }
                    None => break,
                }
            }
        }
    }
}

// ── Single-container log reconnection (Phase 3) ────────────────────────────

/// Find a container ID by stack+service, trying label match then name fallback.
async fn find_container_id(
    docker: &docker::DockerClient,
    stack: &str,
    service: &str,
) -> Option<String> {
    let containers = docker::container_list(docker, Some(stack))
        .await
        .unwrap_or_default();

    // Prefer exact service_name match
    if let Some(c) = containers.iter().find(|c| c.service_name == service) {
        return Some(c.name.clone());
    }

    // Fallback: name contains service
    if let Some(c) = containers.iter().find(|c| c.name.contains(service)) {
        return Some(c.name.clone());
    }

    // Last resort: conventional name
    let conventional = format!("{}-{}-1", stack, service);
    Some(conventional)
}

/// Context for a single-container log reconnection loop.
struct ContainerLogCtx<'a> {
    docker: &'a docker::DockerClient,
    stack: &'a str,
    service: &'a str,
    handle: &'a TerminalHandle,
    term_name: &'a str,
    event_bus: &'a EventBus,
}

/// Stream single-container logs with EventBus-driven reconnection on restart.
async fn run_container_log_loop(
    ctx: &ContainerLogCtx<'_>,
    initial_container: &str,
    cancel: CancellationToken,
) {
    let mut container_name = initial_container.to_string();
    let mut last_seen_nano: Option<i64> = None;
    let mut first = true;

    loop {
        if cancel.is_cancelled() {
            return;
        }

        // First connect: tail=100, no filter. Reconnect: since=<seconds> + nano filter.
        let (tail, since, filter_nano) = if first {
            ("100".to_string(), None, None)
        } else {
            let since_secs = last_seen_nano.map(|n| (n / 1_000_000_000) as i32);
            (String::new(), since_secs, last_seen_nano)
        };
        first = false;

        let nanos = stream_single_container_to_terminal(StreamLogOpts {
            docker: ctx.docker,
            container_name: &container_name,
            handle: ctx.handle,
            term_name: ctx.term_name,
            cancel: cancel.clone(),
            tail: &tail,
            since,
            filter_nano,
        }).await;
        if let Some(n) = nanos {
            last_seen_nano = Some(n);
        }

        if cancel.is_cancelled() {
            return;
        }

        // Log stream ended — subscribe to EventBus and wait for restart
        let mut event_rx = ctx.event_bus.subscribe();
        let stack_owned = ctx.stack.to_string();
        let service_owned = ctx.service.to_string();

        loop {
            tokio::select! {
                () = cancel.cancelled() => return,
                event = event_rx.recv() => {
                    match event {
                        Ok(evt) if evt.project == stack_owned
                            && evt.service == service_owned
                            && evt.event_type == "container"
                            && evt.action == "start" =>
                        {
                            // Update container name (may have changed after recreate)
                            if let Some(name) = find_container_id(ctx.docker, ctx.stack, ctx.service).await {
                                container_name = name;
                            }
                            break; // Reconnect
                        }
                        Err(RecvError::Lagged(_)) => {}
                        Err(RecvError::Closed) => return,
                        _ => {}
                    }
                }
            }
        }
    }
}

/// Stream single-container logs by container name with EventBus-driven reconnection.
async fn run_container_log_by_name_loop(
    docker: &docker::DockerClient,
    container_name: &str,
    handle: &TerminalHandle,
    term_name: &str,
    event_bus: &EventBus,
    cancel: CancellationToken,
) {
    let mut last_seen_nano: Option<i64> = None;
    let mut first = true;

    loop {
        if cancel.is_cancelled() {
            return;
        }

        // First connect: tail=100, no filter. Reconnect: since=<seconds> + nano filter.
        let (tail, since, filter_nano) = if first {
            ("100".to_string(), None, None)
        } else {
            let since_secs = last_seen_nano.map(|n| (n / 1_000_000_000) as i32);
            (String::new(), since_secs, last_seen_nano)
        };
        first = false;

        let nanos = stream_single_container_to_terminal(StreamLogOpts {
            docker,
            container_name,
            handle,
            term_name,
            cancel: cancel.clone(),
            tail: &tail,
            since,
            filter_nano,
        }).await;
        if let Some(n) = nanos {
            last_seen_nano = Some(n);
        }

        if cancel.is_cancelled() {
            return;
        }

        // Log stream ended — wait for restart
        let mut event_rx = event_bus.subscribe();
        let cname = container_name.to_string();

        loop {
            tokio::select! {
                () = cancel.cancelled() => return,
                event = event_rx.recv() => {
                    match event {
                        Ok(evt) if evt.event_type == "container"
                            && (evt.name == cname || evt.container_id == cname) =>
                        {
                            if evt.action == "start" {
                                break; // Reconnect
                            }
                        }
                        Err(RecvError::Lagged(_)) => {}
                        Err(RecvError::Closed) => return,
                        _ => {}
                    }
                }
            }
        }
    }
}

/// Options for streaming a single container's logs.
struct StreamLogOpts<'a> {
    docker: &'a docker::DockerClient,
    container_name: &'a str,
    handle: &'a TerminalHandle,
    term_name: &'a str,
    cancel: CancellationToken,
    tail: &'a str,
    since: Option<i32>,
    /// Skip lines with timestamps <= this value (dedup on reconnect).
    filter_nano: Option<i64>,
}

/// Stream logs from a single container into a terminal buffer via the actor.
/// Returns the last timestamp seen (nanoseconds since epoch), or `None` if no log lines were received.
async fn stream_single_container_to_terminal(opts: StreamLogOpts<'_>) -> Option<i64> {
    let log_opts = docker::ContainerLogsOpts {
        follow: true,
        stdout: true,
        stderr: true,
        timestamps: true,
        tail: opts.tail.to_string(),
        since: opts.since,
    };

    let mut stream = std::pin::pin!(docker::container_logs(opts.docker, opts.container_name, log_opts));
    let mut last_seen_nano: Option<i64> = None;

    loop {
        tokio::select! {
            () = opts.cancel.cancelled() => return last_seen_nano,
            item = stream.next() => {
                match item {
                    Some(Ok(output)) => {
                        let data = output.into_bytes();
                        if !data.is_empty() {
                            // Parse timestamp from the line for nano-precision tracking.
                            let text = String::from_utf8_lossy(&data);
                            let (ts_str, content) = split_timestamp(&text);
                            if !ts_str.is_empty()
                                && let Some(nanos) = parse_timestamp_nanos(ts_str)
                            {
                                // Nano filter: skip lines within the already-seen window
                                if let Some(boundary) = opts.filter_nano
                                    && nanos <= boundary
                                {
                                    continue;
                                }
                                last_seen_nano = Some(nanos);
                                // Send as binary-framed entry (timestamp in binary, message without Docker timestamp)
                                let msg = format!("{content}\n");
                                write_log_entry(opts.handle, opts.term_name, nanos, msg.as_bytes());
                            } else {
                                // No parseable timestamp — send with nanos=0
                                write_log_entry(opts.handle, opts.term_name, 0, &data);
                            }
                        }
                    }
                    Some(Err(e)) => {
                        debug!(container = %opts.container_name, "log stream error: {e}");
                        break;
                    }
                    None => break,
                }
            }
        }
    }

    last_seen_nano
}

/// Parse an RFC3339Nano timestamp string into nanoseconds since epoch.
/// Handles "2025-01-15T00:00:09.800000000Z" and similar formats.
fn parse_timestamp_nanos(ts: &str) -> Option<i64> {
    if ts.len() < 19 { return None; }
    let year: i32 = ts[0..4].parse().ok()?;
    let month: u32 = ts[5..7].parse().ok()?;
    let day: u32 = ts[8..10].parse().ok()?;
    let hour: u32 = ts[11..13].parse().ok()?;
    let min: u32 = ts[14..16].parse().ok()?;
    let sec: u32 = ts[17..19].parse().ok()?;

    let days = days_from_civil(year, month, day);
    let secs = days * 86400 + (hour * 3600 + min * 60 + sec) as i64;

    // Parse fractional seconds (nanoseconds)
    let mut nanos: i64 = 0;
    let bytes = ts.as_bytes();
    if bytes.len() > 19 && bytes[19] == b'.' {
        let frac_start = 20;
        // Find end of fractional digits (before 'Z', '+', '-', or end)
        let mut frac_end = frac_start;
        while frac_end < bytes.len() && bytes[frac_end].is_ascii_digit() {
            frac_end += 1;
        }
        let frac_len = frac_end - frac_start;
        if frac_len > 0 {
            // Parse up to 9 digits, pad with zeros
            let digits = frac_len.min(9);
            let frac_val: i64 = ts[frac_start..frac_start + digits].parse().unwrap_or(0);
            // Scale to nanoseconds (multiply by 10^(9-digits))
            let scale = [1_000_000_000, 100_000_000, 10_000_000, 1_000_000, 100_000, 10_000, 1_000, 100, 10, 1];
            nanos = frac_val * scale[digits];
        }
    }

    Some(secs * 1_000_000_000 + nanos)
}

/// Convert a civil date to days since epoch (1970-01-01).
fn days_from_civil(year: i32, month: u32, day: u32) -> i64 {
    let y = if month <= 2 { year as i64 - 1 } else { year as i64 };
    let era = if y >= 0 { y } else { y - 399 } / 400;
    let yoe = (y - era * 400) as u64;
    let m = month as u64;
    let doy = (153 * (if m > 2 { m - 3 } else { m + 9 }) + 2) / 5 + day as u64 - 1;
    let doe = yoe * 365 + yoe / 4 - yoe / 100 + doy;
    era * 146097 + doe as i64 - 719468
}
