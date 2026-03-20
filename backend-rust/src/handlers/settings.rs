use std::collections::BTreeMap;
use std::sync::Arc;

use serde::Serialize;

use crate::broadcast::WsControlMsg;
use crate::ws::protocol::{ErrorResponse, OkResponse};
use crate::ws::WsServer;

use super::{parse_args, arg_object, AppState};

const GLOBAL_ENV_DEFAULT: &str = "# VARIABLE=value #comment";

pub fn register(ws: &mut WsServer, state: Arc<AppState>) {
    // getSettings
    ws.handle_with_state("getSettings", state.clone(), |state, conn, msg| async move {
        let uid = state.check_login(&conn, &msg).await;
        if uid == 0 { return; }

        let mut settings = match state.get_all_settings() {
            Ok(s) => s,
            Err(e) => {
                if let Some(id) = msg.id {
                    conn.send_ack(id, ErrorResponse::new(format!("Database error: {e}"))).await;
                }
                return;
            }
        };

        // Filter out jwtSecret
        settings.remove("jwtSecret");

        // Add globalENV from disk
        let global_env = read_global_env(&state.config.stacks_dir);
        settings.insert("globalENV".to_string(), global_env);

        #[derive(Serialize)]
        struct SettingsResponse { ok: bool, data: BTreeMap<String, String> }

        if let Some(id) = msg.id {
            conn.send_ack(id, SettingsResponse { ok: true, data: settings }).await;
        }
    });

    // setSettings
    ws.handle_with_state("setSettings", state.clone(), |state, conn, msg| async move {
        let uid = state.check_login(&conn, &msg).await;
        if uid == 0 { return; }

        let args = parse_args(&msg);
        let data: std::collections::HashMap<String, serde_json::Value> =
            match arg_object(&args, 0) {
                Some(d) => d,
                None => {
                    if let Some(id) = msg.id {
                        conn.send_ack(id, ErrorResponse::new("Invalid arguments")).await;
                    }
                    return;
                }
            };

        for (key, value) in &data {
            if key == "jwtSecret" { continue; }

            if key == "globalENV" {
                let content = value.as_str().unwrap_or("");
                write_global_env(&state.config.stacks_dir, content);
                continue;
            }

            // Convert value to string for storage
            let str_val = match value {
                serde_json::Value::String(s) => s.clone(),
                serde_json::Value::Bool(b) => if *b { "1".into() } else { "0".into() },
                serde_json::Value::Number(n) => n.to_string(),
                _ => value.to_string(),
            };

            if let Err(e) = state.set_setting(key, &str_val) {
                if let Some(id) = msg.id {
                    conn.send_ack(id, ErrorResponse::new(format!("Failed to save setting: {e}"))).await;
                }
                return;
            }
        }

        if let Some(id) = msg.id {
            conn.send_ack(id, OkResponse {
                ok: true,
                msg: Some("Saved".into()),
                token: None,
            }).await;
        }
    });

    // disconnectOtherSocketClients
    ws.handle_with_state("disconnectOtherSocketClients", state.clone(), |state, conn, msg| async move {
        let uid = state.check_login(&conn, &msg).await;
        if uid == 0 { return; }

        let (done_tx, done_rx) = tokio::sync::oneshot::channel();
        let _ = state.ws_control_tx.send(WsControlMsg::DisconnectOthers {
            keep_conn_id: conn.id.clone(),
            done: done_tx,
        }).await;
        let _ = done_rx.await;

        if let Some(id) = msg.id {
            conn.send_ack(id, OkResponse::simple()).await;
        }
    });
}

fn read_global_env(stacks_dir: &str) -> String {
    let path = std::path::Path::new(stacks_dir).join("global.env");
    match std::fs::read_to_string(&path) {
        Ok(content) if !content.is_empty() => content,
        _ => GLOBAL_ENV_DEFAULT.to_string(),
    }
}

fn write_global_env(stacks_dir: &str, content: &str) {
    let path = std::path::Path::new(stacks_dir).join("global.env");

    if content.is_empty() || content == GLOBAL_ENV_DEFAULT {
        let _ = std::fs::remove_file(&path);
    } else {
        let _ = std::fs::write(&path, content);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    // ── read_global_env ─────────────────────────────────────────────────

    #[test]
    fn read_missing_file_returns_default() {
        let tmp = tempfile::tempdir().unwrap();
        let result = read_global_env(tmp.path().to_str().unwrap());
        assert_eq!(result, GLOBAL_ENV_DEFAULT);
    }

    #[test]
    fn read_existing_file_returns_content() {
        let tmp = tempfile::tempdir().unwrap();
        std::fs::write(tmp.path().join("global.env"), "MY_VAR=hello").unwrap();
        let result = read_global_env(tmp.path().to_str().unwrap());
        assert_eq!(result, "MY_VAR=hello");
    }

    // ── write_global_env ────────────────────────────────────────────────

    #[test]
    fn write_creates_file() {
        let tmp = tempfile::tempdir().unwrap();
        write_global_env(tmp.path().to_str().unwrap(), "FOO=bar");
        let content = std::fs::read_to_string(tmp.path().join("global.env")).unwrap();
        assert_eq!(content, "FOO=bar");
    }

    #[test]
    fn write_deletes_when_default() {
        let tmp = tempfile::tempdir().unwrap();
        write_global_env(tmp.path().to_str().unwrap(), "FOO=bar");
        assert!(tmp.path().join("global.env").exists());
        write_global_env(tmp.path().to_str().unwrap(), GLOBAL_ENV_DEFAULT);
        assert!(!tmp.path().join("global.env").exists());
    }
}
