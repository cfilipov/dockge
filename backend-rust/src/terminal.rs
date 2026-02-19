use portable_pty::{native_pty_system, CommandBuilder, PtySize};
use std::collections::HashMap;
use std::io::{Read, Write};
use std::path::Path;
use std::sync::Arc;
use tokio::sync::{broadcast, mpsc, watch, RwLock};
use tracing::debug;

const BUFFER_LIMIT: usize = 100;

pub const TERMINAL_COLS: u16 = 105;
pub const TERMINAL_ROWS: u16 = 10;
#[allow(dead_code)]
pub const PROGRESS_TERMINAL_ROWS: u16 = 8;
#[allow(dead_code)]
pub const COMBINED_TERMINAL_COLS: u16 = 58;
#[allow(dead_code)]
pub const COMBINED_TERMINAL_ROWS: u16 = 20;

/// Global terminal manager singleton
pub static TERMINAL_MANAGER: std::sync::LazyLock<TerminalManager> =
    std::sync::LazyLock::new(TerminalManager::new);

/// Naming helpers matching the Node.js common/util-common.ts
pub fn get_compose_terminal_name(endpoint: &str, stack: &str) -> String {
    format!("compose-{}-{}", endpoint, stack)
}

pub fn get_combined_terminal_name(endpoint: &str, stack: &str) -> String {
    format!("combined-{}-{}", endpoint, stack)
}

pub fn get_container_exec_terminal_name(endpoint: &str, stack: &str, container: &str, index: u32) -> String {
    format!("container-exec-{}-{}-{}-{}", endpoint, stack, container, index)
}

pub fn get_container_log_name(endpoint: &str, _stack: &str, container: &str) -> String {
    format!("container-log-{}-{}", endpoint, container)
}

/// Commands sent to the PTY command thread
enum PtyCommand {
    Write(Vec<u8>),
    Resize(u16, u16),
    Close,
}

/// A single PTY-backed terminal
#[allow(dead_code)]
pub struct PtyTerminal {
    pub name: String,
    buffer: RwLock<Vec<String>>,
    cmd_tx: std::sync::mpsc::Sender<PtyCommand>,
    output_tx: broadcast::Sender<(String, String)>,
    exit_rx: watch::Receiver<Option<i32>>,
    pub is_interactive: bool,
}

impl PtyTerminal {
    pub async fn push_buffer(&self, data: &str) {
        let mut buffer = self.buffer.write().await;
        buffer.push(data.to_string());
        if buffer.len() > BUFFER_LIMIT {
            buffer.remove(0);
        }
    }

    pub async fn get_buffer(&self) -> String {
        let buffer = self.buffer.read().await;
        buffer.join("")
    }

    pub fn write_input(&self, data: &[u8]) -> bool {
        self.cmd_tx.send(PtyCommand::Write(data.to_vec())).is_ok()
    }

    pub fn resize(&self, rows: u16, cols: u16) -> bool {
        self.cmd_tx.send(PtyCommand::Resize(rows, cols)).is_ok()
    }

    pub fn close(&self) {
        let _ = self.cmd_tx.send(PtyCommand::Close);
    }

    pub fn subscribe(&self) -> broadcast::Receiver<(String, String)> {
        self.output_tx.subscribe()
    }

    /// Wait for the process to exit, returns exit code
    pub async fn wait_for_exit(&self) -> Option<i32> {
        let mut rx = self.exit_rx.clone();
        // Wait until the value changes to Some
        loop {
            if let Some(code) = *rx.borrow() {
                return Some(code);
            }
            if rx.changed().await.is_err() {
                return None;
            }
        }
    }
}

/// Terminal manager — global singleton managing all active terminals
pub struct TerminalManager {
    terminals: RwLock<HashMap<String, Arc<PtyTerminal>>>,
}

impl TerminalManager {
    pub fn new() -> Self {
        Self {
            terminals: RwLock::new(HashMap::new()),
        }
    }

    /// Spawn a PTY terminal process
    fn spawn_pty_inner(
        name: String,
        file: &str,
        args: &[String],
        cwd: &Path,
        rows: u16,
        cols: u16,
        is_interactive: bool,
    ) -> Result<Arc<PtyTerminal>, String> {
        let pty_system = native_pty_system();
        let pair = pty_system
            .openpty(PtySize {
                rows,
                cols,
                pixel_width: 0,
                pixel_height: 0,
            })
            .map_err(|e| format!("Failed to open PTY: {}", e))?;

        let mut cmd = CommandBuilder::new(file);
        cmd.args(args);
        cmd.cwd(cwd);

        let child = pair
            .slave
            .spawn_command(cmd)
            .map_err(|e| format!("Failed to spawn command: {}", e))?;

        // We're done with the slave side
        drop(pair.slave);

        let reader = pair
            .master
            .try_clone_reader()
            .map_err(|e| format!("Failed to clone PTY reader: {}", e))?;

        // Channels
        let (output_async_tx, mut output_async_rx) = mpsc::channel::<Vec<u8>>(256);
        let (cmd_tx, cmd_rx) = std::sync::mpsc::channel::<PtyCommand>();
        let (exit_tx, exit_rx) = watch::channel::<Option<i32>>(None);
        let (broadcast_tx, _) = broadcast::channel::<(String, String)>(256);

        let terminal = Arc::new(PtyTerminal {
            name: name.clone(),
            buffer: RwLock::new(Vec::new()),
            cmd_tx,
            output_tx: broadcast_tx.clone(),
            exit_rx,
            is_interactive,
        });

        // --- Reader thread (blocking read → tokio mpsc) ---
        std::thread::spawn(move || {
            let mut reader = reader;
            let mut buf = [0u8; 4096];
            loop {
                match reader.read(&mut buf) {
                    Ok(0) => break, // EOF
                    Ok(n) => {
                        if output_async_tx.blocking_send(buf[..n].to_vec()).is_err() {
                            break;
                        }
                    }
                    Err(_) => break,
                }
            }
        });

        // --- Command thread (owns writer + master + child, handles Write/Resize/Close) ---
        // Uses recv_timeout + try_wait to detect natural child exit without blocking forever.
        {
            let master = pair.master;
            let mut writer = master
                .take_writer()
                .map_err(|e| format!("Failed to take PTY writer: {}", e))?;

            std::thread::spawn(move || {
                let mut child = child;
                let poll_interval = std::time::Duration::from_millis(100);
                loop {
                    match cmd_rx.recv_timeout(poll_interval) {
                        Ok(PtyCommand::Write(data)) => {
                            if writer.write_all(&data).is_err() {
                                break;
                            }
                        }
                        Ok(PtyCommand::Resize(rows, cols)) => {
                            let _ = master.resize(PtySize {
                                rows,
                                cols,
                                pixel_width: 0,
                                pixel_height: 0,
                            });
                        }
                        Ok(PtyCommand::Close) => {
                            drop(writer);
                            let _ = child.kill();
                            let code = child
                                .wait()
                                .ok()
                                .map(|s| s.exit_code() as i32)
                                .unwrap_or(-1);
                            let _ = exit_tx.send(Some(code));
                            return;
                        }
                        Err(std::sync::mpsc::RecvTimeoutError::Timeout) => {
                            // Check if child process exited naturally
                            if let Ok(Some(status)) = child.try_wait() {
                                let _ = exit_tx.send(Some(status.exit_code() as i32));
                                return;
                            }
                        }
                        Err(std::sync::mpsc::RecvTimeoutError::Disconnected) => {
                            // Channel closed — the PtyTerminal was dropped
                            drop(writer);
                            let code = child
                                .wait()
                                .ok()
                                .map(|s| s.exit_code() as i32)
                                .unwrap_or(-1);
                            let _ = exit_tx.send(Some(code));
                            return;
                        }
                    }
                }
                // If we broke out of the loop (write error), clean up
                let code = child
                    .wait()
                    .ok()
                    .map(|s| s.exit_code() as i32)
                    .unwrap_or(-1);
                let _ = exit_tx.send(Some(code));
            });
        }

        // --- Async task: receive from reader thread → push to buffer + broadcast ---
        {
            let terminal_weak = Arc::downgrade(&terminal);
            let name = name.clone();
            tokio::spawn(async move {
                while let Some(data) = output_async_rx.recv().await {
                    let text = String::from_utf8_lossy(&data).to_string();
                    if let Some(terminal) = terminal_weak.upgrade() {
                        terminal.push_buffer(&text).await;
                    }
                    // Always broadcast even if terminal ref is gone (subscribers may still exist)
                    let _ = broadcast_tx.send((name.clone(), text));
                }
            });
        }

        Ok(terminal)
    }

    /// Execute a command in a new PTY terminal, stream output, wait for exit.
    /// Returns the exit code.
    pub async fn exec(
        &self,
        name: &str,
        file: &str,
        args: &[String],
        cwd: &Path,
        output_tx: broadcast::Sender<(String, String)>,
    ) -> Result<i32, String> {
        if self.has(name).await {
            return Err("Another operation is already running, please try again later.".to_string());
        }

        let terminal = Self::spawn_pty_inner(
            name.to_string(),
            file,
            args,
            cwd,
            TERMINAL_ROWS,
            TERMINAL_COLS,
            false,
        )?;

        {
            let mut terminals = self.terminals.write().await;
            terminals.insert(name.to_string(), terminal.clone());
        }

        // Show command being executed
        let cmd_display = format!("{} {}", file, args.join(" "));
        let cmd_line = format!("\x1b[90m$ {}\x1b[0m\r\n", cmd_display);
        terminal.push_buffer(&cmd_line).await;
        let _ = output_tx.send((name.to_string(), cmd_line));

        // Forward terminal's broadcast to the caller's output_tx
        let mut rx = terminal.subscribe();
        let caller_tx = output_tx.clone();
        let forward_task = tokio::spawn(async move {
            while let Ok(msg) = rx.recv().await {
                if caller_tx.send(msg).is_err() {
                    break;
                }
            }
        });

        // Wait for exit
        let exit_code = terminal.wait_for_exit().await.unwrap_or(-1);

        forward_task.abort();

        // Remove from map
        self.remove(name).await;

        Ok(exit_code)
    }

    /// Spawn a persistent terminal (interactive exec, logs, main shell).
    /// Returns the terminal. Caller should subscribe to its output.
    pub async fn spawn_persistent(
        &self,
        name: &str,
        file: &str,
        args: &[String],
        cwd: &Path,
        rows: u16,
        cols: u16,
    ) -> Result<Arc<PtyTerminal>, String> {
        // If already exists, return existing
        {
            let terminals = self.terminals.read().await;
            if let Some(t) = terminals.get(name) {
                return Ok(t.clone());
            }
        }

        let terminal = Self::spawn_pty_inner(
            name.to_string(),
            file,
            args,
            cwd,
            rows,
            cols,
            true,
        )?;

        {
            let mut terminals = self.terminals.write().await;
            terminals.insert(name.to_string(), terminal.clone());
        }

        // Auto-remove when process exits
        let name_owned = name.to_string();
        let terminal_weak = Arc::downgrade(&terminal);
        let manager_terminals = &self.terminals as *const RwLock<HashMap<String, Arc<PtyTerminal>>>;
        // SAFETY: TERMINAL_MANAGER is a global static, so the reference outlives the task
        let terminals_ref = unsafe { &*manager_terminals };
        tokio::spawn(async move {
            if let Some(t) = terminal_weak.upgrade() {
                let _ = t.wait_for_exit().await;
            }
            let mut terminals = terminals_ref.write().await;
            terminals.remove(&name_owned);
            debug!("Persistent terminal {} exited and removed", name_owned);
        });

        Ok(terminal)
    }

    #[allow(dead_code)]
    pub async fn get(&self, name: &str) -> Option<Arc<PtyTerminal>> {
        let terminals = self.terminals.read().await;
        terminals.get(name).cloned()
    }

    pub async fn get_buffer(&self, name: &str) -> String {
        let terminals = self.terminals.read().await;
        if let Some(t) = terminals.get(name) {
            t.get_buffer().await
        } else {
            String::new()
        }
    }

    pub async fn write_input(&self, name: &str, data: &[u8]) -> bool {
        let terminals = self.terminals.read().await;
        if let Some(t) = terminals.get(name) {
            t.write_input(data)
        } else {
            false
        }
    }

    pub async fn resize(&self, name: &str, rows: u16, cols: u16) -> bool {
        let terminals = self.terminals.read().await;
        if let Some(t) = terminals.get(name) {
            t.resize(rows, cols)
        } else {
            false
        }
    }

    pub async fn remove(&self, name: &str) {
        let mut terminals = self.terminals.write().await;
        if let Some(t) = terminals.remove(name) {
            t.close();
        }
    }

    pub async fn has(&self, name: &str) -> bool {
        let terminals = self.terminals.read().await;
        terminals.contains_key(name)
    }
}
