//! Terminal manager: buffer management, writer fan-out, PTY execution.
//!
//! Mirrors the Go terminal package. Each Terminal is either a Pipe (read-only
//! log stream) or a PTY (interactive shell). Output is buffered (max 64KB,
//! rolling 32KB window) and fanned out to all registered writers.

use std::collections::HashMap;
use std::io::Write;
use std::sync::Arc;

use std::sync::Mutex;
use tokio::sync::mpsc;
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
pub struct Terminal {
    #[allow(dead_code)] // Used for debug logging when handlers are wired up
    pub name: String,
    pub terminal_type: TerminalType,
    buffer: Vec<u8>,
    pub writers: HashMap<String, WriterFn>,
    closed: bool,
    pub cancel: Option<tokio_util::sync::CancellationToken>,
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
    pub fn write_data(&mut self, data: &[u8]) {
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
    pub fn join_and_get_buffer(&mut self, key: String, writer: WriterFn) -> Vec<u8> {
        self.writers.insert(key, writer);
        self.buffer.clone()
    }

    pub fn remove_writer(&mut self, key: &str) {
        self.writers.remove(key);
    }

    pub fn writer_count(&self) -> usize {
        self.writers.len()
    }

    pub fn is_closed(&self) -> bool {
        self.closed
    }

    pub fn has_cancel(&self) -> bool {
        self.cancel.is_some()
    }

    /// Send input data to the PTY stdin.
    pub fn input(&self, data: &[u8]) {
        if let Some(ref tx) = self.pty_input_tx {
            let _ = tx.send(PtyCommand::Input(data.to_vec()));
        }
    }

    /// Resize the PTY window.
    pub fn resize(&self, rows: u16, cols: u16) {
        if let Some(ref tx) = self.pty_input_tx {
            let _ = tx.send(PtyCommand::Resize { rows, cols });
        }
    }

    /// Close the terminal: cancel streams, clear writers.
    pub fn close(&mut self) {
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

/// Terminal manager: tracks all active terminals.
///
/// Lock ordering: always acquire outer `terminals` mutex first, then inner
/// per-terminal mutex. Never hold an inner lock while acquiring the outer one.
pub struct Manager {
    pub terminals: Mutex<HashMap<String, Arc<Mutex<Terminal>>>>,
}

impl Manager {
    pub fn new() -> Self {
        Self {
            terminals: Mutex::new(HashMap::new()),
        }
    }

    /// Get an existing terminal by name.
    pub fn get(&self, name: &str) -> Option<Arc<Mutex<Terminal>>> {
        self.terminals.lock().unwrap().get(name).cloned()
    }

    /// Get or create a pipe terminal.
    pub fn get_or_create(&self, name: &str) -> Arc<Mutex<Terminal>> {
        let mut terminals = self.terminals.lock().unwrap();
        terminals
            .entry(name.to_string())
            .or_insert_with(|| Arc::new(Mutex::new(Terminal::new(name.to_string(), TerminalType::Pipe))))
            .clone()
    }

    /// Create a fresh terminal, closing the old one asynchronously if it exists.
    pub fn create(&self, name: &str, typ: TerminalType) -> Arc<Mutex<Terminal>> {
        let mut terminals = self.terminals.lock().unwrap();
        if let Some(old) = terminals.remove(name) {
            // Close old terminal asynchronously
            let old = old.clone();
            tokio::spawn(async move {
                old.lock().unwrap().close();
            });
        }
        let term = Arc::new(Mutex::new(Terminal::new(name.to_string(), typ)));
        terminals.insert(name.to_string(), term.clone());
        term
    }

    /// Create a fresh terminal, carrying over writers from the old one.
    pub fn recreate(&self, name: &str, typ: TerminalType) -> Arc<Mutex<Terminal>> {
        let mut terminals = self.terminals.lock().unwrap();
        let mut new_term = Terminal::new(name.to_string(), typ);

        if let Some(old) = terminals.remove(name) {
            let mut old_guard = old.lock().unwrap();
            // Carry over writers
            std::mem::swap(&mut new_term.writers, &mut old_guard.writers);
            old_guard.close();
        }

        let term = Arc::new(Mutex::new(new_term));
        terminals.insert(name.to_string(), term.clone());
        term
    }

    /// Remove a terminal immediately.
    #[allow(dead_code)] // Handler wiring pending
    pub fn remove(&self, name: &str) {
        let mut terminals = self.terminals.lock().unwrap();
        if let Some(term) = terminals.remove(name) {
            term.lock().unwrap().close();
        }
    }

    /// Remove writer from a specific terminal and clean up if orphaned.
    #[allow(dead_code)] // Handler wiring pending
    pub fn remove_writer_and_cleanup(&self, term_name: &str, writer_key: &str) {
        let terminals = self.terminals.lock().unwrap();
        if let Some(term) = terminals.get(term_name) {
            let mut t = term.lock().unwrap();
            t.remove_writer(writer_key);
            // For pipe terminals with cancel and no writers, cancel immediately
            if t.writer_count() == 0 && t.terminal_type == TerminalType::Pipe && t.has_cancel() {
                t.close();
            }
        }
    }

    /// Remove writer from all terminals (on connection disconnect).
    pub fn remove_writer_from_all(&self, writer_key: &str) {
        let terminals = self.terminals.lock().unwrap();
        for term in terminals.values() {
            let mut t = term.lock().unwrap();
            t.remove_writer(writer_key);
            // Immediately close orphaned pipe terminals with cancel
            if t.writer_count() == 0 && t.terminal_type == TerminalType::Pipe && t.has_cancel() {
                t.close();
            }
        }
    }

    /// Start a PTY command and attach it to a terminal.
    /// Returns a CancellationToken that cancels the PTY when dropped.
    pub fn start_pty(
        &self,
        term: &Arc<Mutex<Terminal>>,
        cmd_name: &str,
        cmd_args: &[&str],
        working_dir: Option<&str>,
    ) -> Result<tokio_util::sync::CancellationToken, Box<dyn std::error::Error + Send + Sync>> {
        use portable_pty::PtySize;

        let setup = open_pty(cmd_name, cmd_args, working_dir)?;
        let mut pty_reader = setup.reader;
        let mut master_writer = setup.writer;
        let master_for_resize = setup.master;

        let cancel = tokio_util::sync::CancellationToken::new();

        // Input channel for PTY stdin
        let (input_tx, mut input_rx) = mpsc::unbounded_channel::<PtyCommand>();

        {
            let mut t = term.lock().unwrap();
            t.cancel = Some(cancel.clone());
            t.pty_input_tx = Some(input_tx);
        }

        // Reader task: PTY stdout → terminal buffer
        let term_reader = term.clone();
        let cancel_reader = cancel.clone();
        std::thread::spawn(move || {
            let mut buf = [0u8; 4096];
            loop {
                if cancel_reader.is_cancelled() {
                    break;
                }
                match pty_reader.read(&mut buf) {
                    Ok(0) => break,
                    Ok(n) => {
                        term_reader.lock().unwrap().write_data(&buf[..n]);
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

    /// Run a PTY command synchronously (blocks until completion).
    /// Used for compose actions that need to stream output.
    #[allow(dead_code)] // Handler wiring pending
    pub async fn run_pty(
        &self,
        term: &Arc<Mutex<Terminal>>,
        cmd_name: &str,
        cmd_args: &[&str],
        working_dir: Option<&str>,
    ) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        let setup = open_pty(cmd_name, cmd_args, working_dir)?;
        let mut pty_reader = setup.reader;
        let mut child = setup.child;
        // Drop master + writer to signal EOF to the child's stdin
        drop(setup.writer);
        drop(setup.master);

        // Read PTY output in a background thread
        let term_clone = term.clone();
        let reader_handle = std::thread::spawn(move || {
            let mut buf = [0u8; 4096];
            loop {
                match pty_reader.read(&mut buf) {
                    Ok(0) => break,
                    Ok(n) => {
                        term_clone.lock().unwrap().write_data(&buf[..n]);
                    }
                    Err(e) => {
                        if e.kind() != std::io::ErrorKind::Interrupted {
                            break;
                        }
                    }
                }
            }
        });

        // Wait for child to exit
        let status = tokio::task::spawn_blocking(move || child.wait()).await??;

        // Wait for reader to finish
        let _ = reader_handle.join();

        if status.success() {
            Ok(())
        } else {
            Err(format!("command exited with status: {:?}", status).into())
        }
    }
}

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
