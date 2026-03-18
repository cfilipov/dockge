use std::sync::Arc;

use crate::broadcast::WsControlMsg;
use crate::ws::conn::Conn;
use crate::ws::protocol::{ClientMessage, ErrorResponse, OkResponse};
use crate::ws::WsServer;

use super::{parse_args, arg_object, AppState};

const GLOBAL_ENV_DEFAULT: &str = "# VARIABLE=value #comment";

pub fn register(ws: &mut WsServer, state: Arc<AppState>) {
    // getSettings
    {
        let state = state.clone();
        ws.handle("getSettings", move |conn: Arc<Conn>, msg: ClientMessage| {
            let state = state.clone();
            async move {
                let uid = state.check_login(&conn, &msg).await;
                if uid == 0 { return; }

                let mut settings = state.get_all_settings();

                // Filter out jwtSecret
                settings.remove("jwtSecret");

                // Add globalENV from disk
                let global_env = read_global_env(&state.config.stacks_dir);
                settings.insert("globalENV".to_string(), global_env);

                if let Some(id) = msg.id {
                    conn.send_ack(id, serde_json::json!({
                        "ok": true,
                        "data": settings,
                    })).await;
                }
            }
        });
    }

    // setSettings
    {
        let state = state.clone();
        ws.handle("setSettings", move |conn: Arc<Conn>, msg: ClientMessage| {
            let state = state.clone();
            async move {
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

                    state.set_setting(key, &str_val);
                }

                if let Some(id) = msg.id {
                    conn.send_ack(id, OkResponse {
                        ok: true,
                        msg: Some("Saved".into()),
                        token: None,
                    }).await;
                }
            }
        });
    }

    // disconnectOtherSocketClients
    {
        let state = state.clone();
        ws.handle("disconnectOtherSocketClients", move |conn: Arc<Conn>, msg: ClientMessage| {
            let state = state.clone();
            async move {
                let uid = state.check_login(&conn, &msg).await;
                if uid == 0 { return; }

                let (done_tx, done_rx) = tokio::sync::oneshot::channel();
                let _ = state.ws_control_tx.send(WsControlMsg::DisconnectOthers {
                    keep_conn_id: conn.id.clone(),
                    done: done_tx,
                }).await;
                let _ = done_rx.await;

                if let Some(id) = msg.id {
                    conn.send_ack(id, OkResponse { ok: true, msg: None, token: None }).await;
                }
            }
        });
    }
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
