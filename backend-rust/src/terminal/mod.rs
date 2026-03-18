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
#[allow(dead_code)]
pub struct Terminal {
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

#[allow(dead_code)]
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

    /// Get the current buffer contents.
    pub fn buffer(&self) -> &[u8] {
        &self.buffer
    }

    /// Atomically add a writer and return the current buffer.
    pub fn join_and_get_buffer(&mut self, key: String, writer: WriterFn) -> Vec<u8> {
        self.writers.insert(key, writer);
        self.buffer.clone()
    }

    pub fn add_writer(&mut self, key: String, writer: WriterFn) {
        self.writers.insert(key, writer);
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
#[allow(dead_code)]
pub struct Manager {
    pub terminals: Mutex<HashMap<String, Arc<Mutex<Terminal>>>>,
}

#[allow(dead_code)]
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
    pub fn remove(&self, name: &str) {
        let mut terminals = self.terminals.lock().unwrap();
        if let Some(term) = terminals.remove(name) {
            term.lock().unwrap().close();
        }
    }

    /// Schedule terminal removal after a delay. Cancels if terminal was recreated.
    pub fn remove_after(&self, name: &str, delay: std::time::Duration) {
        let name = name.to_string();
        let term_ptr = {
            let terminals = self.terminals.lock().unwrap();
            match terminals.get(&name) {
                Some(t) => Arc::as_ptr(t) as usize,
                None => return,
            }
        };
        let manager = self as *const Manager as usize;
        tokio::spawn(async move {
            tokio::time::sleep(delay).await;
            // Safety: we only read, and the Manager outlives this task in practice
            let mgr = unsafe { &*(manager as *const Manager) };
            let mut terminals = mgr.terminals.lock().unwrap();
            if let Some(existing) = terminals.get(&name) {
                // Only remove if it's the same terminal instance (not recreated)
                if Arc::as_ptr(existing) as usize == term_ptr
                    && let Some(term) = terminals.remove(&name)
                {
                    term.lock().unwrap().close();
                }
            }
        });
    }

    /// Remove writer from a specific terminal and clean up if orphaned.
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
        drop(pair.slave); // Close slave side — child has it

        // Get reader and writer from master before moving it
        let mut pty_reader = pair.master.try_clone_reader()?;
        let mut master_writer = pair.master.take_writer()?;
        let master_for_resize = pair.master;

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
    pub async fn run_pty(
        &self,
        term: &Arc<Mutex<Terminal>>,
        cmd_name: &str,
        cmd_args: &[&str],
        working_dir: Option<&str>,
    ) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
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

        let mut child = pair.slave.spawn_command(cmd)?;
        drop(pair.slave);

        let mut pty_reader = pair.master.try_clone_reader()?;
        drop(pair.master);

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

    /// Periodic cleanup: remove closed terminals with no writers.
    pub fn cleanup_completed(&self) {
        let mut terminals = self.terminals.lock().unwrap();
        terminals.retain(|_, term| {
            let t = term.lock().unwrap();
            !(t.is_closed() && t.writer_count() == 0)
        });
    }

    /// Spawn a background cleanup loop.
    pub fn spawn_cleanup_loop(&'static self, cancel: tokio_util::sync::CancellationToken) {
        tokio::spawn(async move {
            let mut interval = tokio::time::interval(std::time::Duration::from_secs(60));
            loop {
                tokio::select! {
                    () = cancel.cancelled() => return,
                    _ = interval.tick() => {
                        self.cleanup_completed();
                    }
                }
            }
        });
    }
}
