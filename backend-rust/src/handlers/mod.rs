pub mod auth;
pub mod docker;
pub mod image_updates;
pub mod logs;
pub mod service;
pub mod settings;
pub mod stack;
pub mod terminal;

use std::sync::Arc;
use std::sync::atomic::AtomicBool;
use std::collections::BTreeMap;

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
    /// Set to true once the background image update checker has completed its first run.
    pub image_check_complete: AtomicBool,
}

impl AppState {
    /// Check that the connection is authenticated.
    /// Returns the user ID or sends an error ack and returns 0.
    ///
    /// In `--no-auth` mode, connections are auto-authenticated at connect time
    /// (user_id set to 1), so this returns 1 without special handling — matching
    /// the Go backend's behavior where logout clears user_id back to 0.
    pub fn check_login(&self, conn: &Conn, msg: &ClientMessage) -> i32 {
        let uid = conn.user_id();
        if uid == 0
            && let Some(id) = msg.id
        {
            conn.send_ack(id, ErrorResponse::new("Not logged in"));
        }
        uid
    }

    /// Get all settings from the database.
    pub fn get_all_settings(&self) -> Result<BTreeMap<String, String>, redb::Error> {
        use redb::ReadableTable;
        let read_txn = self.db.begin_read()?;
        let table = read_txn.open_table(db::SETTINGS_TABLE)?;
        let mut result = BTreeMap::new();
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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ws::protocol::ClientMessage;

    fn make_msg(args_json: Option<&str>) -> ClientMessage {
        let args = args_json.map(|s| RawValue::from_string(s.to_string()).unwrap());
        ClientMessage {
            id: Some(1),
            event: "test".to_string(),
            args,
        }
    }

    // ── parse_args ──────────────────────────────────────────────────────

    #[test]
    fn parse_args_none() {
        let msg = make_msg(None);
        assert!(parse_args(&msg).is_empty());
    }

    #[test]
    fn parse_args_empty_array() {
        let msg = make_msg(Some("[]"));
        assert!(parse_args(&msg).is_empty());
    }

    #[test]
    fn parse_args_string_array() {
        let msg = make_msg(Some(r#"["hello","world"]"#));
        let args = parse_args(&msg);
        assert_eq!(args.len(), 2);
    }

    #[test]
    fn parse_args_mixed_types() {
        let msg = make_msg(Some(r#"["hello",42,true]"#));
        let args = parse_args(&msg);
        assert_eq!(args.len(), 3);
    }

    #[test]
    fn parse_args_invalid_json() {
        let msg = make_msg(Some(r#""not an array""#));
        assert!(parse_args(&msg).is_empty());
    }

    // ── arg_string ──────────────────────────────────────────────────────

    #[test]
    fn arg_string_valid() {
        let msg = make_msg(Some(r#"["hello","world"]"#));
        let args = parse_args(&msg);
        assert_eq!(arg_string(&args, 0), "hello");
        assert_eq!(arg_string(&args, 1), "world");
    }

    #[test]
    fn arg_string_out_of_bounds() {
        let msg = make_msg(Some(r#"["hello"]"#));
        let args = parse_args(&msg);
        assert_eq!(arg_string(&args, 5), "");
    }

    #[test]
    fn arg_string_non_string_value() {
        let msg = make_msg(Some(r#"[42]"#));
        let args = parse_args(&msg);
        assert_eq!(arg_string(&args, 0), "");
    }

    // ── arg_object ──────────────────────────────────────────────────────

    #[test]
    fn arg_object_valid() {
        #[derive(serde::Deserialize, Debug, PartialEq)]
        struct Opts { name: String }

        let msg = make_msg(Some(r#"[{"name":"test"}]"#));
        let args = parse_args(&msg);
        let result: Option<Opts> = arg_object(&args, 0);
        assert_eq!(result, Some(Opts { name: "test".to_string() }));
    }

    #[test]
    fn arg_object_out_of_bounds() {
        let msg = make_msg(Some(r#"[]"#));
        let args = parse_args(&msg);
        let result: Option<serde_json::Value> = arg_object(&args, 0);
        assert!(result.is_none());
    }

    #[test]
    fn arg_object_wrong_shape() {
        #[derive(serde::Deserialize)]
        #[allow(dead_code)]
        struct Opts { name: String }

        let msg = make_msg(Some(r#"["not an object"]"#));
        let args = parse_args(&msg);
        let result: Option<Opts> = arg_object(&args, 0);
        assert!(result.is_none());
    }
}
