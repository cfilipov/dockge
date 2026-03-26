use std::sync::Arc;

use futures_util::StreamExt;
use serde::Deserialize;
use tokio::sync::broadcast::error::RecvError;
use tokio::task::JoinSet;
use tokio_util::sync::CancellationToken;
use tracing::{debug, warn};

use crate::broadcast::eventbus::EventBus;
use crate::docker;
use crate::ws::conn::Conn;
use crate::ws::protocol::{ErrorResponse, OkResponse};
use crate::ws::WsServer;

use super::{arg_object, parse_args, AppState};

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

// ── Timestamp parsing ───────────────────────────────────────────────────────

/// Split a Docker log line with timestamp prefix into (timestamp, rest).
fn split_timestamp(raw: &str) -> (&str, &str) {
    if let Some(pos) = raw[..raw.len().min(35)].find(' ') {
        let ts = &raw[..pos];
        if ts.starts_with(|c: char| c.is_ascii_digit()) {
            return (ts, &raw[pos + 1..]);
        }
    }
    ("", raw)
}

/// Parse an RFC3339Nano timestamp string into nanoseconds since epoch.
fn parse_timestamp_nanos(ts: &str) -> Option<i64> {
    if ts.len() < 19 {
        return None;
    }
    let year: i32 = ts[0..4].parse().ok()?;
    let month: u32 = ts[5..7].parse().ok()?;
    let day: u32 = ts[8..10].parse().ok()?;
    let hour: u32 = ts[11..13].parse().ok()?;
    let min: u32 = ts[14..16].parse().ok()?;
    let sec: u32 = ts[17..19].parse().ok()?;

    let days = days_from_civil(year, month, day);
    let secs = days * 86400 + (hour * 3600 + min * 60 + sec) as i64;

    let mut nanos: i64 = 0;
    let bytes = ts.as_bytes();
    if bytes.len() > 19 && bytes[19] == b'.' {
        let frac_start = 20;
        let mut frac_end = frac_start;
        while frac_end < bytes.len() && bytes[frac_end].is_ascii_digit() {
            frac_end += 1;
        }
        let frac_len = frac_end - frac_start;
        if frac_len > 0 {
            let digits = frac_len.min(9);
            let frac_val: i64 = ts[frac_start..frac_start + digits].parse().unwrap_or(0);
            let scale = [
                1_000_000_000, 100_000_000, 10_000_000, 1_000_000, 100_000, 10_000, 1_000, 100,
                10, 1,
            ];
            nanos = frac_val * scale[digits];
        }
    }

    Some(secs * 1_000_000_000 + nanos)
}

fn days_from_civil(year: i32, month: u32, day: u32) -> i64 {
    let y = if month <= 2 {
        year as i64 - 1
    } else {
        year as i64
    };
    let era = if y >= 0 { y } else { y - 399 } / 400;
    let yoe = (y - era * 400) as u64;
    let m = month as u64;
    let doy = (153 * (if m > 2 { m - 3 } else { m + 9 }) + 2) / 5 + day as u64 - 1;
    let doe = yoe * 365 + yoe / 4 - yoe / 100 + doy;
    era * 146097 + doe as i64 - 719468
}

// ── JSON log line helper ────────────────────────────────────────────────────

/// Send a logData event to the client. Returns false if the connection is dead.
fn send_log_line(conn: &Conn, nanos: i64, line: &str) -> bool {
    #[derive(serde::Serialize)]
    struct LogData<'a> {
        ts: i64,
        line: &'a str,
    }
    conn.send_event("logData", LogData { ts: nanos, line })
}

// ── Handler registration ────────────────────────────────────────────────────

pub fn register(ws: &mut WsServer, state: Arc<AppState>) {
    // subscribeLogs
    ws.handle_with_state("subscribeLogs", state.clone(), |state, conn, msg| async move {
        let uid = state.check_login(&conn, &msg);
        if uid == 0 {
            return;
        }

        let args = parse_args(&msg);
        #[derive(Deserialize)]
        struct SubscribeArgs {
            #[serde(rename = "type")]
            log_type: String,
            #[serde(default)]
            stack: String,
            #[serde(default)]
            service: String,
            #[serde(default)]
            container: String,
        }

        let sub_args: SubscribeArgs = match arg_object(&args, 0) {
            Some(a) => a,
            None => {
                if let Some(id) = msg.id {
                    conn.send_ack(id, ErrorResponse::new("Invalid arguments"));
                }
                return;
            }
        };

        // Cancel any existing log subscription for this connection
        let token = CancellationToken::new();
        conn.set_subscription("logs", String::new(), token.clone());

        if let Some(id) = msg.id {
            conn.send_ack(id, OkResponse::simple());
        }

        match sub_args.log_type.as_str() {
            "combined" => {
                if sub_args.stack.is_empty() {
                    return;
                }
                let docker = state.docker.clone();
                let event_bus = state.event_bus.clone();
                let stack = sub_args.stack;
                let conn = conn.clone();
                tokio::spawn(async move {
                    stream_combined_logs(&docker, &stack, &conn, &event_bus, token).await;
                });
            }
            "container-log" => {
                if sub_args.stack.is_empty() || sub_args.service.is_empty() {
                    return;
                }
                let docker = state.docker.clone();
                let event_bus = state.event_bus.clone();
                let stack = sub_args.stack;
                let service = sub_args.service;
                let container = if sub_args.container.is_empty() {
                    format!("{}-{}-1", stack, service)
                } else {
                    sub_args.container
                };
                let conn = conn.clone();
                tokio::spawn(async move {
                    stream_container_log(
                        &docker, &stack, &service, &container, &conn, &event_bus, token,
                    )
                    .await;
                });
            }
            "container-log-by-name" => {
                if sub_args.container.is_empty() {
                    return;
                }
                let docker = state.docker.clone();
                let event_bus = state.event_bus.clone();
                let container = sub_args.container;
                let conn = conn.clone();
                tokio::spawn(async move {
                    stream_container_log_by_name(&docker, &container, &conn, &event_bus, token)
                        .await;
                });
            }
            other => {
                warn!("unknown log type: {other}");
            }
        }
    });

    // unsubscribeLogs
    ws.handle_with_state("unsubscribeLogs", state, |_state, conn, msg| async move {
        conn.cancel_subscription("logs", "");
        if let Some(id) = msg.id {
            conn.send_ack(id, OkResponse::simple());
        }
    });
}

// ── Combined log streaming ──────────────────────────────────────────────────

/// A log line with its original timestamp for sorting.
struct TsLine {
    ts: String,
    nanos: i64,
    display: String,
}

async fn stream_combined_logs(
    docker: &docker::DockerClient,
    stack: &str,
    conn: &Conn,
    event_bus: &EventBus,
    cancel: CancellationToken,
) {
    let containers = docker::container_list(docker, Some(stack))
        .await
        .unwrap_or_default();

    let mut max_len = containers
        .iter()
        .map(|c| c.service_name.len())
        .max()
        .unwrap_or(0);

    let mut color_map: std::collections::HashMap<String, usize> =
        std::collections::HashMap::new();
    for (i, c) in containers.iter().enumerate() {
        color_map.entry(c.service_name.clone()).or_insert(i);
    }

    // Phase 1: Historical logs with merge-sort
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
                                display: format!("{prefix}{content}"),
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

        all_lines.sort_by(|a, b| a.ts.cmp(&b.ts));

        for line in &all_lines {
            if cancel.is_cancelled() {
                return;
            }
            if !send_log_line(conn, line.nanos, &line.display) {
                return;
            }
        }
    }

    // Phase 2: Live follow with EventBus-driven reconnection
    {
        let (line_tx, mut line_rx) = tokio::sync::mpsc::channel::<(i64, String)>(256);
        let mut tasks = JoinSet::new();

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

        let mut event_rx = event_bus.subscribe();
        let stack_owned = stack.to_string();

        loop {
            tokio::select! {
                () = cancel.cancelled() => return,
                line = line_rx.recv() => {
                    match line {
                        Some((nanos, text)) => {
                            if !send_log_line(conn, nanos, &text) {
                                return;
                            }
                        }
                        None => break,
                    }
                }
                event = event_rx.recv() => {
                    match event {
                        Ok(evt) if evt.project == stack_owned && evt.event_type == "container" => {
                            if evt.action == "start" {
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
                        Err(RecvError::Lagged(_)) => {}
                        Err(RecvError::Closed) => break,
                        _ => {}
                    }
                }
                Some(_) = tasks.join_next() => {}
            }
        }
    }
}

/// Follow a single container's logs (live), sending prefixed lines to `tx`.
async fn follow_container_logs(
    docker: &docker::DockerClient,
    container_name: &str,
    prefix: &str,
    tx: &tokio::sync::mpsc::Sender<(i64, String)>,
    cancel: &CancellationToken,
    since: Option<i32>,
) {
    let opts = docker::ContainerLogsOpts {
        follow: true,
        stdout: true,
        stderr: true,
        timestamps: true,
        tail: if since.is_some() {
            String::new()
        } else {
            "0".to_string()
        },
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
                            let display = format!("{prefix}{content}");
                            if tx.send((nanos, display)).await.is_err() {
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

// ── Single-container log streaming ──────────────────────────────────────────

/// Find a container ID by stack+service.
async fn find_container_id(
    docker: &docker::DockerClient,
    stack: &str,
    service: &str,
) -> Option<String> {
    let containers = docker::container_list(docker, Some(stack))
        .await
        .unwrap_or_default();

    if let Some(c) = containers.iter().find(|c| c.service_name == service) {
        return Some(c.name.clone());
    }
    if let Some(c) = containers.iter().find(|c| c.name.contains(service)) {
        return Some(c.name.clone());
    }
    Some(format!("{}-{}-1", stack, service))
}

async fn stream_container_log(
    docker: &docker::DockerClient,
    stack: &str,
    service: &str,
    initial_container: &str,
    conn: &Conn,
    event_bus: &EventBus,
    cancel: CancellationToken,
) {
    let mut container_name = initial_container.to_string();
    let mut last_seen_nano: Option<i64> = None;
    let mut first = true;

    loop {
        if cancel.is_cancelled() {
            return;
        }

        let (tail, since, filter_nano) = if first {
            ("100".to_string(), None, None)
        } else {
            let since_secs = last_seen_nano.map(|n| (n / 1_000_000_000) as i32);
            (String::new(), since_secs, last_seen_nano)
        };
        first = false;

        let nanos =
            stream_single_container(conn, docker, &container_name, &tail, since, filter_nano, &cancel)
                .await;
        if let Some(n) = nanos {
            last_seen_nano = Some(n);
        }

        if cancel.is_cancelled() {
            return;
        }

        // Wait for restart event
        let mut event_rx = event_bus.subscribe();
        let stack_owned = stack.to_string();
        let service_owned = service.to_string();

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
                            if let Some(name) = find_container_id(docker, stack, service).await {
                                container_name = name;
                            }
                            break;
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

async fn stream_container_log_by_name(
    docker: &docker::DockerClient,
    container_name: &str,
    conn: &Conn,
    event_bus: &EventBus,
    cancel: CancellationToken,
) {
    let mut last_seen_nano: Option<i64> = None;
    let mut first = true;

    loop {
        if cancel.is_cancelled() {
            return;
        }

        let (tail, since, filter_nano) = if first {
            ("100".to_string(), None, None)
        } else {
            let since_secs = last_seen_nano.map(|n| (n / 1_000_000_000) as i32);
            (String::new(), since_secs, last_seen_nano)
        };
        first = false;

        let nanos =
            stream_single_container(conn, docker, container_name, &tail, since, filter_nano, &cancel)
                .await;
        if let Some(n) = nanos {
            last_seen_nano = Some(n);
        }

        if cancel.is_cancelled() {
            return;
        }

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
                                break;
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

/// Stream logs from a single container, sending JSON logData events.
/// Returns the last timestamp seen (nanos), or None if no lines were received.
async fn stream_single_container(
    conn: &Conn,
    docker: &docker::DockerClient,
    container_name: &str,
    tail: &str,
    since: Option<i32>,
    filter_nano: Option<i64>,
    cancel: &CancellationToken,
) -> Option<i64> {
    let log_opts = docker::ContainerLogsOpts {
        follow: true,
        stdout: true,
        stderr: true,
        timestamps: true,
        tail: tail.to_string(),
        since,
    };

    let mut stream = std::pin::pin!(docker::container_logs(docker, container_name, log_opts));
    let mut last_seen_nano: Option<i64> = None;

    loop {
        tokio::select! {
            () = cancel.cancelled() => return last_seen_nano,
            item = stream.next() => {
                match item {
                    Some(Ok(output)) => {
                        let data = output.into_bytes();
                        if !data.is_empty() {
                            let text = String::from_utf8_lossy(&data);
                            for line in text.lines() {
                                if line.is_empty() { continue; }
                                let (ts_str, content) = split_timestamp(line);
                                if !ts_str.is_empty()
                                    && let Some(nanos) = parse_timestamp_nanos(ts_str) {
                                        if let Some(boundary) = filter_nano
                                            && nanos <= boundary { continue; }
                                        last_seen_nano = Some(nanos);
                                        if !send_log_line(conn, nanos, content) {
                                            return last_seen_nano;
                                        }
                                        continue;
                                    }
                                // Non-timestamped line
                                if !send_log_line(conn, 0, line) {
                                    return last_seen_nano;
                                }
                            }
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

    last_seen_nano
}
