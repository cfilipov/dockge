use std::collections::{HashMap, VecDeque};
use std::path::{Path, PathBuf};
use std::sync::Arc;
use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader};
use tokio::process::Command;
use tokio::sync::{broadcast, mpsc, RwLock};

const BUFFER_LIMIT: usize = 100;

#[allow(dead_code)]
pub const TERMINAL_COLS: u16 = 105;
#[allow(dead_code)]
pub const TERMINAL_ROWS: u16 = 10;
#[allow(dead_code)]
pub const PROGRESS_TERMINAL_ROWS: u16 = 8;
#[allow(dead_code)]
pub const COMBINED_TERMINAL_COLS: u16 = 58;
#[allow(dead_code)]
pub const COMBINED_TERMINAL_ROWS: u16 = 20;

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

/// Terminal manager — global singleton managing all active terminals
pub struct TerminalManager {
    terminals: RwLock<HashMap<String, Arc<TerminalInstance>>>,
}

impl TerminalManager {
    pub fn new() -> Self {
        Self {
            terminals: RwLock::new(HashMap::new()),
        }
    }

    #[allow(dead_code)]
    pub async fn get(&self, name: &str) -> Option<Arc<TerminalInstance>> {
        let terminals = self.terminals.read().await;
        terminals.get(name).cloned()
    }

    #[allow(dead_code)]
    pub async fn get_or_create(
        &self,
        name: &str,
        file: &str,
        args: &[String],
        cwd: &Path,
        enable_keep_alive: bool,
    ) -> Arc<TerminalInstance> {
        {
            let terminals = self.terminals.read().await;
            if let Some(t) = terminals.get(name) {
                return t.clone();
            }
        }

        let terminal = Arc::new(TerminalInstance::new(
            name.to_string(),
            file.to_string(),
            args.to_vec(),
            cwd.to_path_buf(),
            enable_keep_alive,
        ));

        let mut terminals = self.terminals.write().await;
        terminals.insert(name.to_string(), terminal.clone());
        terminal
    }

    pub async fn remove(&self, name: &str) {
        let mut terminals = self.terminals.write().await;
        terminals.remove(name);
    }

    pub async fn has(&self, name: &str) -> bool {
        let terminals = self.terminals.read().await;
        terminals.contains_key(name)
    }

    #[allow(dead_code)]
    pub async fn count(&self) -> usize {
        let terminals = self.terminals.read().await;
        terminals.len()
    }

    /// Execute a command in a new terminal, stream output, wait for exit.
    /// Returns the exit code.
    pub async fn exec(
        &self,
        name: &str,
        file: &str,
        args: &[String],
        cwd: &Path,
        output_tx: broadcast::Sender<(String, String)>, // (terminal_name, data)
    ) -> Result<i32, String> {
        if self.has(name).await {
            return Err("Another operation is already running, please try again later.".to_string());
        }

        let terminal = Arc::new(TerminalInstance::new(
            name.to_string(),
            file.to_string(),
            args.to_vec(),
            cwd.to_path_buf(),
            false,
        ));

        {
            let mut terminals = self.terminals.write().await;
            terminals.insert(name.to_string(), terminal.clone());
        }

        // Show command being executed
        let cmd_display = format!("{} {}", file, args.join(" "));
        let cmd_line = format!("\x1b[90m$ {}\x1b[0m\r\n", cmd_display);
        terminal.push_buffer(&cmd_line).await;
        let _ = output_tx.send((name.to_string(), cmd_line));

        let exit_code = terminal.run(output_tx).await;

        // Remove from map
        self.remove(name).await;

        exit_code
    }
}

/// A single terminal instance
#[allow(dead_code)]
pub struct TerminalInstance {
    pub name: String,
    pub file: String,
    pub args: Vec<String>,
    pub cwd: PathBuf,
    pub enable_keep_alive: bool,
    pub buffer: RwLock<VecDeque<String>>,
    pub input_tx: RwLock<Option<mpsc::Sender<String>>>,
    pub started: RwLock<bool>,
}

impl TerminalInstance {
    pub fn new(
        name: String,
        file: String,
        args: Vec<String>,
        cwd: PathBuf,
        enable_keep_alive: bool,
    ) -> Self {
        Self {
            name,
            file,
            args,
            cwd,
            enable_keep_alive,
            buffer: RwLock::new(VecDeque::with_capacity(BUFFER_LIMIT)),
            input_tx: RwLock::new(None),
            started: RwLock::new(false),
        }
    }

    pub async fn push_buffer(&self, data: &str) {
        let mut buffer = self.buffer.write().await;
        buffer.push_back(data.to_string());
        if buffer.len() > BUFFER_LIMIT {
            buffer.pop_front();
        }
    }

    #[allow(dead_code)]
    pub async fn get_buffer(&self) -> String {
        let buffer = self.buffer.read().await;
        buffer.iter().cloned().collect()
    }

    #[allow(dead_code)]
    pub async fn write_input(&self, data: &str) {
        let tx = self.input_tx.read().await;
        if let Some(ref tx) = *tx {
            let _ = tx.send(data.to_string()).await;
        }
    }

    /// Run the process, streaming output to the broadcast sender.
    /// Returns the exit code.
    pub async fn run(
        &self,
        output_tx: broadcast::Sender<(String, String)>,
    ) -> Result<i32, String> {
        let mut already_started = self.started.write().await;
        if *already_started {
            return Err("Terminal already started".to_string());
        }
        *already_started = true;
        drop(already_started);

        let mut cmd = Command::new(&self.file);
        cmd.args(&self.args)
            .current_dir(&self.cwd)
            .stdout(std::process::Stdio::piped())
            .stderr(std::process::Stdio::piped())
            .stdin(std::process::Stdio::piped());

        let mut child = cmd.spawn().map_err(|e| format!("Failed to spawn: {}", e))?;

        let stdout = child.stdout.take();
        let stderr = child.stderr.take();
        let stdin = child.stdin.take();

        // Set up input channel for interactive terminals
        let (input_sender, mut input_receiver) = mpsc::channel::<String>(32);
        {
            let mut tx = self.input_tx.write().await;
            *tx = Some(input_sender);
        }

        let name = self.name.clone();
        let output_tx2 = output_tx.clone();
        let name2 = name.clone();

        // Stdout reader
        let _buf_ref = &self.buffer;
        let stdout_handle = tokio::spawn({
            let name = name.clone();
            let output_tx = output_tx.clone();
            async move {
                if let Some(stdout) = stdout {
                    let reader = BufReader::new(stdout);
                    let mut lines = reader.lines();
                    while let Ok(Some(line)) = lines.next_line().await {
                        let data = format!("{}\r\n", line);
                        let _ = output_tx.send((name.clone(), data.clone()));
                    }
                }
            }
        });

        // Stderr reader
        let stderr_handle = tokio::spawn({
            let name = name2.clone();
            let output_tx = output_tx2;
            async move {
                if let Some(stderr) = stderr {
                    let reader = BufReader::new(stderr);
                    let mut lines = reader.lines();
                    while let Ok(Some(line)) = lines.next_line().await {
                        let data = format!("{}\r\n", line);
                        let _ = output_tx.send((name.clone(), data.clone()));
                    }
                }
            }
        });

        // Stdin writer
        let stdin_handle = tokio::spawn(async move {
            if let Some(mut stdin) = stdin {
                while let Some(data) = input_receiver.recv().await {
                    if stdin.write_all(data.as_bytes()).await.is_err() {
                        break;
                    }
                }
            }
        });

        let _ = tokio::join!(stdout_handle, stderr_handle);

        let status = child.wait().await.map_err(|e| format!("Wait failed: {}", e))?;
        let exit_code = status.code().unwrap_or(-1);

        // Clean up input channel
        drop(stdin_handle);
        {
            let mut tx = self.input_tx.write().await;
            *tx = None;
        }

        Ok(exit_code)
    }
}
