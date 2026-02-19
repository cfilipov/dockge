use crate::handlers::auth::{check_login, send_info};
use crate::models::settings as settings_model;
use crate::models::user::User;
use crate::socket_args::SocketArgs;
use crate::state::AppState;
use serde_json::{json, Value};
use socketioxide::extract::{Data, SocketRef, AckSender};
use std::sync::Arc;
use tracing::warn;

/// Register settings-related socket event handlers
pub fn register(socket: &SocketRef, state: Arc<AppState>) {
    // getSettings
    {
        let state = state.clone();
        socket.on("getSettings", move |socket: SocketRef, ack: AckSender| {
            let state = state.clone();
            tokio::spawn(async move {
                let result = handle_get_settings(&state, &socket).await;
                ack.send(&result).ok();
            });
        });
    }

    // setSettings
    {
        let state = state.clone();
        socket.on("setSettings", move |socket: SocketRef, Data(args): Data<SocketArgs>, ack: AckSender| {
            let state = state.clone();
            tokio::spawn(async move {
                let data = Value::Array(args.0);
                let result = handle_set_settings(&state, &socket, &data).await;
                ack.send(&result).ok();
            });
        });
    }

    // disconnectOtherSocketClients
    {
        let state = state.clone();
        socket.on("disconnectOtherSocketClients", move |socket: SocketRef| {
            let state = state.clone();
            tokio::spawn(async move {
                let user_id = match check_login(&socket) {
                    Some(id) => id,
                    None => return,
                };
                let socket_id = socket.id.to_string();
                crate::handlers::auth::disconnect_other_clients(&state, &socket_id, Some(user_id)).await;
            });
        });
    }

    // composerize
    {
        let state = state.clone();
        socket.on("composerize", move |socket: SocketRef, Data(args): Data<SocketArgs>, ack: AckSender| {
            let state = state.clone();
            tokio::spawn(async move {
                let data = Value::Array(args.0);
                let result = handle_composerize(&state, &socket, &data).await;
                ack.send(&result).ok();
            });
        });
    }
}

async fn handle_get_settings(state: &Arc<AppState>, socket: &SocketRef) -> Value {
    if check_login(socket).is_none() {
        return json!({ "ok": false, "msg": "Not logged in" });
    }

    match settings_model::get_settings(&state.db, "general").await {
        Ok(mut data) => {
            // Add globalENV
            let global_env_path = state.stacks_dir.join("global.env");
            if global_env_path.exists() {
                let content = tokio::fs::read_to_string(&global_env_path)
                    .await
                    .unwrap_or_else(|_| "# VARIABLE=value #comment".to_string());
                data.insert("globalENV".to_string(), Value::String(content));
            } else {
                data.insert("globalENV".to_string(), Value::String("# VARIABLE=value #comment".to_string()));
            }

            json!({ "ok": true, "data": data })
        }
        Err(e) => json!({ "ok": false, "msg": format!("{}", e) }),
    }
}

async fn handle_set_settings(state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
    if check_login(socket).is_none() {
        return json!({ "ok": false, "msg": "Not logged in" });
    }

    // data is [settings_obj, currentPassword]
    let (settings_data, current_password) = if data.is_array() {
        let arr = data.as_array().unwrap();
        let settings = arr.first().unwrap_or(&Value::Null);
        let password = arr.get(1).and_then(|v| v.as_str()).unwrap_or("");
        (settings.clone(), password.to_string())
    } else {
        (data.clone(), String::new())
    };

    let settings_obj = match settings_data.as_object() {
        Some(obj) => obj.clone(),
        None => return json!({ "ok": false, "msg": "Invalid settings data" }),
    };

    // Check if trying to disable auth
    let current_disabled_auth = settings_model::get(&state.db, "disableAuth")
        .await
        .ok()
        .flatten()
        .and_then(|v| v.as_bool())
        .unwrap_or(false);

    let wants_disable_auth = settings_obj
        .get("disableAuth")
        .and_then(|v| v.as_bool())
        .unwrap_or(false);

    if !current_disabled_auth && wants_disable_auth {
        // Need to verify password
        let user_id = check_login(socket).unwrap();
        let user = match User::find_by_id(&state.db, user_id).await {
            Ok(Some(u)) => u,
            _ => return json!({ "ok": false, "msg": "User not found" }),
        };
        if !user.verify_password(&current_password) {
            return json!({ "ok": false, "msg": "Incorrect password" });
        }
    }

    // Handle globalENV
    let mut save_settings = settings_obj.clone();
    if let Some(global_env) = save_settings.remove("globalENV") {
        let env_str = global_env.as_str().unwrap_or("");
        let global_env_path = state.stacks_dir.join("global.env");

        if !env_str.is_empty() && env_str != "# VARIABLE=value #comment" {
            if let Err(e) = tokio::fs::write(&global_env_path, env_str).await {
                warn!("Failed to write global.env: {}", e);
            }
        } else {
            let _ = tokio::fs::remove_file(&global_env_path).await;
        }
    }

    match settings_model::set_settings(&state.db, "general", &save_settings).await {
        Ok(()) => {
            send_info(state, socket, false).await;
            json!({ "ok": true, "msg": "Saved" })
        }
        Err(e) => json!({ "ok": false, "msg": format!("{}", e) }),
    }
}

async fn handle_composerize(_state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
    if check_login(socket).is_none() {
        return json!({ "ok": false, "msg": "Not logged in" });
    }

    let docker_run_cmd = if data.is_array() {
        data.get(0).and_then(|v| v.as_str()).unwrap_or("")
    } else {
        data.as_str().unwrap_or("")
    };

    if docker_run_cmd.is_empty() {
        return json!({ "ok": false, "msg": "dockerRunCommand must be a string" });
    }

    // Shell out to composerize (npm package, needs to be installed)
    // Try using npx composerize
    match tokio::process::Command::new("npx")
        .args(["composerize", docker_run_cmd])
        .output()
        .await
    {
        Ok(output) if output.status.success() => {
            let mut result = String::from_utf8_lossy(&output.stdout).to_string();
            // Remove the first line "name: <project name>"
            if let Some(newline_pos) = result.find('\n') {
                let first_line = &result[..newline_pos];
                if first_line.starts_with("name:") {
                    result = result[newline_pos + 1..].to_string();
                }
            }
            json!({ "ok": true, "composeTemplate": result })
        }
        Ok(output) => {
            let stderr = String::from_utf8_lossy(&output.stderr);
            json!({ "ok": false, "msg": format!("composerize failed: {}", stderr) })
        }
        Err(e) => json!({ "ok": false, "msg": format!("Failed to run composerize: {}", e) }),
    }
}
