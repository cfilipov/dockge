//! Terminal manager: buffer management, writer fan-out, PTY execution.
//!
//! Mirrors the Go terminal package. Each Terminal is either a Pipe (read-only
//! log stream) or a PTY (interactive shell). Output is buffered (max 64KB,
//! rolling 32KB window) and fanned out to all registered writers.
//!
//! All mutable state is owned by a single actor task. External code interacts
//! via [`TerminalHandle`], which sends commands over an mpsc channel.

use std::collections::HashMap;
use std::io::Write;

use tokio::sync::{mpsc, oneshot};
use tracing::debug;

const MAX_BUFFER: usize = 64 * 1024;
const KEEP_BUFFER: usize = 32 * 1024;

/// Terminal output writer callback. Receives binary data to send to a client.
pub type WriterFn = Box<dyn Fn(&[u8]) + Send + Sync + 'static>;

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum TerminalType {
    Pipe,
    Pty,
}

/// A single terminal (pipe or PTY) with buffer and fan-out.
/// Private to the actor — external code uses [`TerminalHandle`].
struct Terminal {
    #[allow(dead_code)] // Used for debug logging
    name: String,
    terminal_type: TerminalType,
    buffer: Vec<u8>,
    writers: HashMap<String, WriterFn>,
    closed: bool,
    cancel: Option<tokio_util::sync::CancellationToken>,
    /// For PTY: channel to send input to the PTY stdin writer task.
    pty_input_tx: Option<mpsc::UnboundedSender<PtyCommand>>,
}

enum PtyCommand {
    Input(Vec<u8>),
    Resize { rows: u16, cols: u16 },
}

impl Terminal {
    fn new(name: String, terminal_type: TerminalType) -> Self {
        Self {
            name,
            terminal_type,
            buffer: Vec::with_capacity(4096),
            writers: HashMap::new(),
            closed: false,
            cancel: None,
            pty_input_tx: None,
        }
    }

    /// Write data to the buffer and fan out to all writers.
    fn write_data(&mut self, data: &[u8]) {
        if data.is_empty() || self.closed {
            return;
        }

        // For pipe terminals, normalize bare \n to \r\n
        let normalized;
        let output = if self.terminal_type == TerminalType::Pipe {
            normalized = normalize_newlines(data);
            &normalized
        } else {
            data
        };

        // Append to buffer with rolling window
        self.buffer.extend_from_slice(output);
        if self.buffer.len() > MAX_BUFFER {
            let start = self.buffer.len() - KEEP_BUFFER;
            self.buffer = self.buffer[start..].to_vec();
        }

        // Fan out to all writers
        for writer in self.writers.values() {
            writer(output);
        }
    }

    /// Atomically add a writer and return the current buffer.
    fn join_and_get_buffer(&mut self, key: String, writer: WriterFn) -> Vec<u8> {
        self.writers.insert(key, writer);
        self.buffer.clone()
    }

    fn remove_writer(&mut self, key: &str) {
        self.writers.remove(key);
    }

    fn writer_count(&self) -> usize {
        self.writers.len()
    }

    fn is_closed(&self) -> bool {
        self.closed
    }

    fn has_cancel(&self) -> bool {
        self.cancel.is_some()
    }

    /// Send input data to the PTY stdin.
    fn input(&self, data: &[u8]) {
        if let Some(ref tx) = self.pty_input_tx {
            let _ = tx.send(PtyCommand::Input(data.to_vec()));
        }
    }

    /// Resize the PTY window.
    fn resize(&self, rows: u16, cols: u16) {
        if let Some(ref tx) = self.pty_input_tx {
            let _ = tx.send(PtyCommand::Resize { rows, cols });
        }
    }

    /// Close the terminal: cancel streams, clear writers.
    fn close(&mut self) {
        self.closed = true;
        if let Some(cancel) = self.cancel.take() {
            cancel.cancel();
        }
        self.pty_input_tx = None;
        self.writers.clear();
    }
}

/// Normalize bare \n to \r\n (for pipe terminals).
fn normalize_newlines(data: &[u8]) -> Vec<u8> {
    let mut result = Vec::with_capacity(data.len() + data.len() / 10);
    for (i, &b) in data.iter().enumerate() {
        if b == b'\n' && (i == 0 || data[i - 1] != b'\r') {
            result.push(b'\r');
        }
        result.push(b);
    }
    result
}

// ── Actor commands ──────────────────────────────────────────────────────────

enum TerminalCmd {
    GetOrCreate {
        name: String,
        reply: oneshot::Sender<()>,
    },
    Create {
        name: String,
        typ: TerminalType,
        reply: oneshot::Sender<()>,
    },
    Recreate {
        name: String,
        typ: TerminalType,
        reply: oneshot::Sender<()>,
    },
    IsClosed {
        name: String,
        reply: oneshot::Sender<Option<bool>>,
    },
    JoinAndGetBuffer {
        name: String,
        writer_key: String,
        writer: WriterFn,
        reply: oneshot::Sender<Vec<u8>>,
    },
    RemoveWriterFromAll {
        writer_key: String,
    },
    WriteData {
        name: String,
        data: Vec<u8>,
    },
    InputByWriterKey {
        writer_key: String,
        data: Vec<u8>,
    },
    ResizeByWriterKey {
        writer_key: String,
        rows: u16,
        cols: u16,
    },
    SetCancel {
        name: String,
        cancel: tokio_util::sync::CancellationToken,
    },
    StartPty {
        name: String,
        cmd: String,
        args: Vec<String>,
        working_dir: Option<String>,
        reply: oneshot::Sender<Result<tokio_util::sync::CancellationToken, String>>,
    },
}

// ── TerminalHandle (public API) ─────────────────────────────────────────────

/// Channel-based handle to the terminal actor. Clone-cheap (just an mpsc sender).
#[derive(Clone)]
pub struct TerminalHandle {
    tx: mpsc::Sender<TerminalCmd>,
}

impl TerminalHandle {
    /// Get or create a pipe terminal by name.
    pub async fn get_or_create(&self, name: &str) {
        let (reply, rx) = oneshot::channel();
        let _ = self.tx.send(TerminalCmd::GetOrCreate {
            name: name.to_string(),
            reply,
        }).await;
        let _ = rx.await;
    }

    /// Create a fresh terminal, closing the old one if it exists.
    pub async fn create(&self, name: &str, typ: TerminalType) {
        let (reply, rx) = oneshot::channel();
        let _ = self.tx.send(TerminalCmd::Create {
            name: name.to_string(),
            typ,
            reply,
        }).await;
        let _ = rx.await;
    }

    /// Create a fresh terminal, carrying over writers from the old one.
    pub async fn recreate(&self, name: &str, typ: TerminalType) {
        let (reply, rx) = oneshot::channel();
        let _ = self.tx.send(TerminalCmd::Recreate {
            name: name.to_string(),
            typ,
            reply,
        }).await;
        let _ = rx.await;
    }

    /// Check if a terminal exists and whether it is closed.
    /// Returns `None` if the terminal doesn't exist.
    pub async fn is_closed(&self, name: &str) -> Option<bool> {
        let (reply, rx) = oneshot::channel();
        let _ = self.tx.send(TerminalCmd::IsClosed {
            name: name.to_string(),
            reply,
        }).await;
        rx.await.ok().flatten()
    }

    /// Register a writer and get the current buffer contents.
    pub async fn join_and_get_buffer(
        &self,
        name: &str,
        writer_key: String,
        writer: WriterFn,
    ) -> Vec<u8> {
        let (reply, rx) = oneshot::channel();
        let _ = self.tx.send(TerminalCmd::JoinAndGetBuffer {
            name: name.to_string(),
            writer_key,
            writer,
            reply,
        }).await;
        rx.await.unwrap_or_default()
    }

    /// Remove a writer from all terminals (fire-and-forget).
    pub fn remove_writer_from_all(&self, writer_key: &str) {
        let _ = self.tx.try_send(TerminalCmd::RemoveWriterFromAll {
            writer_key: writer_key.to_string(),
        });
    }

    /// Write data to a terminal's buffer (fire-and-forget).
    pub fn write_data(&self, name: &str, data: Vec<u8>) {
        let _ = self.tx.try_send(TerminalCmd::WriteData {
            name: name.to_string(),
            data,
        });
    }

    /// Send input to the terminal associated with a writer key (fire-and-forget).
    pub fn input_by_writer_key(&self, writer_key: &str, data: Vec<u8>) {
        let _ = self.tx.try_send(TerminalCmd::InputByWriterKey {
            writer_key: writer_key.to_string(),
            data,
        });
    }

    /// Resize the terminal associated with a writer key (fire-and-forget).
    pub fn resize_by_writer_key(&self, writer_key: &str, rows: u16, cols: u16) {
        let _ = self.tx.try_send(TerminalCmd::ResizeByWriterKey {
            writer_key: writer_key.to_string(),
            rows,
            cols,
        });
    }

    /// Set a cancellation token on a terminal (fire-and-forget).
    pub fn set_cancel(&self, name: &str, cancel: tokio_util::sync::CancellationToken) {
        let _ = self.tx.try_send(TerminalCmd::SetCancel {
            name: name.to_string(),
            cancel,
        });
    }

    /// Start a PTY command attached to a terminal.
    pub async fn start_pty(
        &self,
        name: &str,
        cmd: &str,
        args: &[&str],
        working_dir: Option<&str>,
    ) -> Result<tokio_util::sync::CancellationToken, String> {
        let (reply, rx) = oneshot::channel();
        let _ = self.tx.send(TerminalCmd::StartPty {
            name: name.to_string(),
            cmd: cmd.to_string(),
            args: args.iter().map(|s| s.to_string()).collect(),
            working_dir: working_dir.map(|s| s.to_string()),
            reply,
        }).await;
        rx.await.map_err(|_| "actor dropped".to_string())?
    }
}

// ── Actor loop ──────────────────────────────────────────────────────────────

/// Spawn the terminal actor. Returns a cloneable handle.
pub fn spawn() -> TerminalHandle {
    let (tx, rx) = mpsc::channel(256);
    let handle = TerminalHandle { tx: tx.clone() };
    tokio::spawn(actor_loop(tx, rx));
    handle
}

async fn actor_loop(
    actor_tx: mpsc::Sender<TerminalCmd>,
    mut rx: mpsc::Receiver<TerminalCmd>,
) {
    let mut terminals: HashMap<String, Terminal> = HashMap::new();
    // writer_key → terminal name for O(1) lookup
    let mut writer_index: HashMap<String, String> = HashMap::new();

    while let Some(cmd) = rx.recv().await {
        match cmd {
            TerminalCmd::GetOrCreate { name, reply } => {
                terminals
                    .entry(name.clone())
                    .or_insert_with(|| Terminal::new(name, TerminalType::Pipe));
                let _ = reply.send(());
            }

            TerminalCmd::Create { name, typ, reply } => {
                if let Some(mut old) = terminals.remove(&name) {
                    // Remove old terminal's writer_index entries
                    let keys: Vec<String> = old.writers.keys().cloned().collect();
                    for k in keys {
                        writer_index.remove(&k);
                    }
                    old.close();
                }
                terminals.insert(name.clone(), Terminal::new(name, typ));
                let _ = reply.send(());
            }

            TerminalCmd::Recreate { name, typ, reply } => {
                let mut new_term = Terminal::new(name.clone(), typ);
                if let Some(mut old) = terminals.remove(&name) {
                    // Carry over writers (writer_index entries stay valid — name unchanged)
                    std::mem::swap(&mut new_term.writers, &mut old.writers);
                    old.close();
                }
                terminals.insert(name, new_term);
                let _ = reply.send(());
            }

            TerminalCmd::IsClosed { name, reply } => {
                let result = terminals.get(&name).map(|t| t.is_closed());
                let _ = reply.send(result);
            }

            TerminalCmd::JoinAndGetBuffer { name, writer_key, writer, reply } => {
                if let Some(term) = terminals.get_mut(&name) {
                    writer_index.insert(writer_key.clone(), name);
                    let buffer = term.join_and_get_buffer(writer_key, writer);
                    let _ = reply.send(buffer);
                } else {
                    let _ = reply.send(Vec::new());
                }
            }

            TerminalCmd::RemoveWriterFromAll { writer_key } => {
                // Use writer_index for O(1) lookup if available
                if let Some(term_name) = writer_index.remove(&writer_key) {
                    if let Some(term) = terminals.get_mut(&term_name) {
                        term.remove_writer(&writer_key);
                        if term.writer_count() == 0
                            && term.terminal_type == TerminalType::Pipe
                            && term.has_cancel()
                        {
                            term.close();
                        }
                    }
                } else {
                    // Fallback: scan all terminals (handles legacy keys not in index)
                    for term in terminals.values_mut() {
                        term.remove_writer(&writer_key);
                        if term.writer_count() == 0
                            && term.terminal_type == TerminalType::Pipe
                            && term.has_cancel()
                        {
                            term.close();
                        }
                    }
                }
            }

            TerminalCmd::WriteData { name, data } => {
                if let Some(term) = terminals.get_mut(&name) {
                    term.write_data(&data);
                }
            }

            TerminalCmd::InputByWriterKey { writer_key, data } => {
                if let Some(term_name) = writer_index.get(&writer_key)
                    && let Some(term) = terminals.get(term_name)
                {
                    term.input(&data);
                }
            }

            TerminalCmd::ResizeByWriterKey { writer_key, rows, cols } => {
                if let Some(term_name) = writer_index.get(&writer_key)
                    && let Some(term) = terminals.get(term_name)
                {
                    term.resize(rows, cols);
                }
            }

            TerminalCmd::SetCancel { name, cancel } => {
                if let Some(term) = terminals.get_mut(&name) {
                    term.cancel = Some(cancel);
                }
            }

            TerminalCmd::StartPty { name, cmd, args, working_dir, reply } => {
                let arg_refs: Vec<&str> = args.iter().map(|s| s.as_str()).collect();
                let wd_ref = working_dir.as_deref();

                let result = match open_pty(&cmd, &arg_refs, wd_ref) {
                    Ok(setup) => {
                        let cancel = tokio_util::sync::CancellationToken::new();
                        let (input_tx, mut input_rx) = mpsc::unbounded_channel::<PtyCommand>();

                        if let Some(term) = terminals.get_mut(&name) {
                            term.cancel = Some(cancel.clone());
                            term.pty_input_tx = Some(input_tx);
                        }

                        // Reader thread: PTY stdout → actor WriteData
                        let mut pty_reader = setup.reader;
                        let cancel_reader = cancel.clone();
                        let actor_tx_clone = actor_tx.clone();
                        let name_clone = name.clone();
                        std::thread::spawn(move || {
                            let mut buf = [0u8; 4096];
                            loop {
                                if cancel_reader.is_cancelled() {
                                    break;
                                }
                                match pty_reader.read(&mut buf) {
                                    Ok(0) => break,
                                    Ok(n) => {
                                        let _ = actor_tx_clone.blocking_send(
                                            TerminalCmd::WriteData {
                                                name: name_clone.clone(),
                                                data: buf[..n].to_vec(),
                                            },
                                        );
                                    }
                                    Err(e) => {
                                        if e.kind() != std::io::ErrorKind::Interrupted {
                                            debug!("pty reader error: {e}");
                                            break;
                                        }
                                    }
                                }
                            }
                        });

                        // Input task: forwards PTY stdin writes and resize commands
                        let mut master_writer = setup.writer;
                        let master_for_resize = setup.master;
                        let cancel_input = cancel.clone();
                        let child = setup.child;
                        tokio::spawn(async move {
                            let _child = child; // Keep child alive until this task ends
                            loop {
                                tokio::select! {
                                    () = cancel_input.cancelled() => break,
                                    cmd = input_rx.recv() => {
                                        match cmd {
                                            Some(PtyCommand::Input(data)) => {
                                                let _ = master_writer.write_all(&data);
                                                let _ = master_writer.flush();
                                            }
                                            Some(PtyCommand::Resize { rows, cols }) => {
                                                use portable_pty::PtySize;
                                                let _ = master_for_resize.resize(PtySize {
                                                    rows, cols,
                                                    pixel_width: 0,
                                                    pixel_height: 0,
                                                });
                                            }
                                            None => break,
                                        }
                                    }
                                }
                            }
                        });

                        Ok(cancel)
                    }
                    Err(e) => Err(e.to_string()),
                };

                let _ = reply.send(result);
            }
        }
    }
}

// ── PTY helpers ─────────────────────────────────────────────────────────────

/// Opened PTY with child process, reader, writer, and master handle.
struct PtySetup {
    child: Box<dyn portable_pty::Child + Send + Sync>,
    reader: Box<dyn std::io::Read + Send>,
    writer: Box<dyn std::io::Write + Send>,
    master: Box<dyn portable_pty::MasterPty + Send>,
}

/// Open a PTY, spawn a command, and return all handles.
fn open_pty(
    cmd_name: &str,
    cmd_args: &[&str],
    working_dir: Option<&str>,
) -> Result<PtySetup, Box<dyn std::error::Error + Send + Sync>> {
    use portable_pty::{CommandBuilder, native_pty_system, PtySize};

    let pty_system = native_pty_system();
    let pair = pty_system.openpty(PtySize {
        rows: 24,
        cols: 80,
        pixel_width: 0,
        pixel_height: 0,
    })?;

    let mut cmd = CommandBuilder::new(cmd_name);
    cmd.args(cmd_args);
    if let Some(dir) = working_dir {
        cmd.cwd(dir);
    }

    let child = pair.slave.spawn_command(cmd)?;
    drop(pair.slave);

    let reader = pair.master.try_clone_reader()?;
    let writer = pair.master.take_writer()?;

    Ok(PtySetup {
        child,
        reader,
        writer,
        master: pair.master,
    })
}
