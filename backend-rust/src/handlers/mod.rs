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
use crate::broadcast::{Broadcaster, DispatchMsg, WsControlMsg};
use crate::config::Config;
use crate::db;
use crate::db::users::UserStore;
use crate::terminal::Manager as TerminalManager;
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
    pub docker: bollard::Docker,
    pub stack_locks: stack::NamedMutex,
    pub has_authenticated: AtomicBool,
    pub terminal_manager: TerminalManager,
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
    pub fn get_all_settings(&self) -> HashMap<String, String> {
        let read_txn = match self.db.begin_read() {
            Ok(t) => t,
            Err(_) => return HashMap::new(),
        };
        let table = match read_txn.open_table(db::SETTINGS_TABLE) {
            Ok(t) => t,
            Err(_) => return HashMap::new(),
        };
        use redb::ReadableTable;
        let mut result = HashMap::new();
        if let Ok(iter) = table.iter() {
            for (k, v) in iter.flatten() {
                result.insert(k.value().to_string(), v.value().to_string());
            }
        }
        result
    }

    /// Set a setting in the database.
    pub fn set_setting(&self, key: &str, value: &str) {
        let write_txn = match self.db.begin_write() {
            Ok(t) => t,
            Err(_) => return,
        };
        {
            let mut table = match write_txn.open_table(db::SETTINGS_TABLE) {
                Ok(t) => t,
                Err(_) => return,
            };
            let _ = table.insert(key, value);
        }
        let _ = write_txn.commit();
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
