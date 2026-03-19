pub mod auth;
pub mod docker;
pub mod image_updates;
pub mod service;
pub mod settings;
pub mod stack;
pub mod terminal;

use std::sync::Arc;
use std::sync::atomic::AtomicBool;
use std::collections::HashMap;

use redb::Database;
use serde_json::value::RawValue;
use tokio::sync::mpsc;
use crate::auth::LoginRateLimiter;
use crate::broadcast::{Broadcaster, DispatchMsg, WsControlMsg, eventbus::EventBus};
use crate::config::Config;
use crate::db;
use crate::db::users::UserStore;
use crate::terminal::TerminalHandle;
use crate::ws::protocol::ClientMessage;
use crate::ws::conn::Conn;
use crate::ws::protocol::ErrorResponse;

/// Shared application state.
///
/// INVARIANT: No Mutex<T> fields. All concurrency is managed through channels
/// (actors), atomics, or internally-synchronized types. Adding a Mutex here
/// would violate the actor-based architecture.
pub struct AppState {
    pub config: Config,
    pub db: Arc<Database>,
    pub users: UserStore,
    pub jwt_secret: String,
    pub need_setup: AtomicBool,
    pub login_limiter: LoginRateLimiter,
    pub broadcaster: Broadcaster,
    pub dispatch_tx: mpsc::Sender<DispatchMsg>,
    pub ws_control_tx: mpsc::Sender<WsControlMsg>,
    pub docker: crate::docker::DockerClient,
    pub stack_locks: stack::NamedMutex,
    pub has_authenticated: AtomicBool,
    pub terminal_manager: TerminalHandle,
    pub event_bus: EventBus,
    /// Set to true once the Docker event watcher has connected to the event stream.
    pub event_watcher_ready: AtomicBool,
}

impl AppState {
    /// Check that the connection is authenticated.
    pub async fn check_login(&self, conn: &Conn, msg: &ClientMessage) -> i32 {
        let uid = conn.user_id();
        if uid == 0
            && let Some(id) = msg.id
        {
            conn.send_ack(id, ErrorResponse::new("Not logged in")).await;
        }
        uid
    }

    /// Get all settings from the database.
    pub fn get_all_settings(&self) -> Result<HashMap<String, String>, redb::Error> {
        use redb::ReadableTable;
        let read_txn = self.db.begin_read()?;
        let table = read_txn.open_table(db::SETTINGS_TABLE)?;
        let mut result = HashMap::new();
        for entry in table.iter()? {
            let (k, v) = entry?;
            result.insert(k.value().to_string(), v.value().to_string());
        }
        Ok(result)
    }

    /// Set a setting in the database.
    pub fn set_setting(&self, key: &str, value: &str) -> Result<(), redb::Error> {
        let write_txn = self.db.begin_write()?;
        {
            let mut table = write_txn.open_table(db::SETTINGS_TABLE)?;
            table.insert(key, value)?;
        }
        write_txn.commit()?;
        Ok(())
    }
}

/// Run a command via PTY, writing status messages to the terminal.
/// Returns `Ok(())` on success (exit code 0), `Err(message)` on failure.
pub(crate) async fn run_pty_to_terminal(
    tm: &TerminalHandle,
    term_name: &str,
    cmd: &str,
    args: &[&str],
    working_dir: Option<&str>,
) -> Result<(), String> {
    match tm.start_pty_and_wait(term_name, cmd, args, working_dir).await {
        Ok((_cancel, done_rx)) => match done_rx.await {
            Ok(Some(0)) | Ok(None) => {
                tm.write_data(term_name, b"\r\n[Done]\r\n".to_vec());
                Ok(())
            }
            Ok(Some(code)) => {
                let msg = format!("\r\n[Error] exit code {code}\r\n");
                tm.write_data(term_name, msg.into_bytes());
                Err(format!("exit code {code}"))
            }
            Err(_) => {
                tm.write_data(term_name, b"\r\n[Error] process lost\r\n".to_vec());
                Err("process lost".into())
            }
        },
        Err(e) => {
            let msg = format!("\r\n[Error] {e}\r\n");
            tm.write_data(term_name, msg.into_bytes());
            Err(e)
        }
    }
}

/// Parse the args JSON array into a Vec of RawValue.
pub fn parse_args(msg: &ClientMessage) -> Vec<Box<RawValue>> {
    match &msg.args {
        Some(raw) => {
            serde_json::from_str::<Vec<Box<RawValue>>>(raw.get()).unwrap_or_default()
        }
        None => Vec::new(),
    }
}

/// Extract a string from args at the given index.
pub fn arg_string(args: &[Box<RawValue>], index: usize) -> String {
    args.get(index)
        .and_then(|v| serde_json::from_str::<String>(v.get()).ok())
        .unwrap_or_default()
}

/// Extract a JSON object from args at the given index.
pub fn arg_object<T: serde::de::DeserializeOwned>(args: &[Box<RawValue>], index: usize) -> Option<T> {
    args.get(index)
        .and_then(|v| serde_json::from_str(v.get()).ok())
}
