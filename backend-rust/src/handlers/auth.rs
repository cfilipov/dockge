use crate::error::error_response;
use crate::models::user::{self, User, check_password_strength, generate_password_hash};
use crate::models::settings;
use crate::state::AppState;
use jsonwebtoken::{decode, DecodingKey, Validation};
use serde_json::{json, Value};
use socketioxide::extract::{Data, SocketRef, AckSender};
use std::sync::Arc;
use tracing::{error, info, warn};

/// Register auth-related socket event handlers
pub fn register(socket: &SocketRef, state: Arc<AppState>) {
    // setup
    {
        let state = state.clone();
        socket.on("setup", move |_socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let state = state.clone();
            tokio::spawn(async move {
                let result = handle_setup(&state, &data).await;
                ack.send(&result).ok();
            });
        });
    }

    // login
    {
        let state = state.clone();
        socket.on("login", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let state = state.clone();
            let socket = socket.clone();
            tokio::spawn(async move {
                let result = handle_login(&state, &socket, &data).await;
                ack.send(&result).ok();
            });
        });
    }

    // loginByToken
    {
        let state = state.clone();
        socket.on("loginByToken", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let state = state.clone();
            let socket = socket.clone();
            tokio::spawn(async move {
                let result = handle_login_by_token(&state, &socket, &data).await;
                ack.send(&result).ok();
            });
        });
    }

    // changePassword
    {
        let state = state.clone();
        socket.on("changePassword", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let state = state.clone();
            let socket = socket.clone();
            tokio::spawn(async move {
                let result = handle_change_password(&state, &socket, &data).await;
                ack.send(&result).ok();
            });
        });
    }

    // needSetup - not an event, but the server sends it on connect
    // getTurnstileSiteKey
    {
        socket.on("getTurnstileSiteKey", move |_socket: SocketRef, ack: AckSender| {
            let site_key = std::env::var("TURNSTILE_SITE_KEY").unwrap_or_default();
            if site_key.is_empty() {
                ack.send(&json!({ "ok": false, "msg": "Turnstile site key is not configured." })).ok();
            } else {
                ack.send(&json!({ "ok": true, "siteKey": site_key })).ok();
            }
        });
    }
}

async fn handle_setup(state: &Arc<AppState>, data: &Value) -> Value {
    let username = data.get(0).and_then(|v| v.as_str()).unwrap_or("");
    let password = data.get(1).and_then(|v| v.as_str()).unwrap_or("");

    if !check_password_strength(password) {
        return error_response(
            "Password is too weak. It should contain alphabetic and numeric characters. It must be at least 6 characters in length."
        );
    }

    match User::count(&state.db).await {
        Ok(count) if count > 0 => {
            return error_response("Dockge has been initialized. If you want to run setup again, please delete the database.");
        }
        Err(e) => return error_response(&format!("Database error: {}", e)),
        _ => {}
    }

    let password_hash = generate_password_hash(password);
    match User::create(&state.db, username, &password_hash).await {
        Ok(_) => {
            let mut need_setup = state.need_setup.write().await;
            *need_setup = false;
            json!({ "ok": true, "msg": "successAdded", "msgi18n": true })
        }
        Err(e) => error_response(&format!("{}", e)),
    }
}

async fn handle_login(state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
    // data is passed as array: [{ username, password, captchaToken?, token? }]
    let login_data = if data.is_array() {
        data.get(0).unwrap_or(data)
    } else {
        data
    };

    let username = login_data.get("username").and_then(|v| v.as_str()).unwrap_or("");
    let password = login_data.get("password").and_then(|v| v.as_str()).unwrap_or("");

    if username.is_empty() || password.is_empty() {
        return json!({ "ok": false, "msg": "authIncorrectCreds", "msgi18n": true });
    }

    // Turnstile CAPTCHA check
    let site_key = std::env::var("TURNSTILE_SITE_KEY").unwrap_or_default();
    let secret_key = std::env::var("TURNSTILE_SECRET_KEY").unwrap_or_default();
    if !site_key.is_empty() && !secret_key.is_empty() {
        let captcha_token = login_data.get("captchaToken").and_then(|v| v.as_str()).unwrap_or("");
        if captcha_token.is_empty() {
            return json!({ "ok": false, "msg": "Invalid CAPTCHA" });
        }
        // In production, we'd verify the token with Cloudflare
        // For now, skip in dev mode
    }

    let user = match User::find_by_username(&state.db, username).await {
        Ok(Some(u)) => u,
        Ok(None) => {
            warn!("Incorrect username or password for user {}", username);
            return json!({ "ok": false, "msg": "authIncorrectCreds", "msgi18n": true });
        }
        Err(e) => return error_response(&format!("{}", e)),
    };

    if !user.verify_password(password) {
        warn!("Incorrect password for user {}", username);
        return json!({ "ok": false, "msg": "authIncorrectCreds", "msgi18n": true });
    }

    // 2FA check
    if user.twofa_status {
        let token_2fa = login_data.get("token").and_then(|v| v.as_str());
        if token_2fa.is_none() {
            return json!({ "tokenRequired": true });
        }
        // 2FA token verification would go here
        // For now, reject as not implemented
        return json!({ "ok": false, "msg": "authInvalidToken", "msgi18n": true });
    }

    let jwt_secret = state.jwt_secret.read().await;
    match user.create_jwt(&jwt_secret) {
        Ok(token) => {
            // Store user ID in socket extensions
            socket.extensions.insert(user.id);
            socket.extensions.insert(user.username.clone());

            info!("Successfully logged in user {}", username);

            // After login: send info, stack list, agent list
            after_login(state, socket).await;

            json!({ "ok": true, "token": token })
        }
        Err(e) => error_response(&format!("{}", e)),
    }
}

async fn handle_login_by_token(state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
    // data is [token_string]
    let token = if data.is_array() {
        data.get(0).and_then(|v| v.as_str()).unwrap_or("")
    } else {
        data.as_str().unwrap_or("")
    };

    if token.is_empty() {
        return json!({ "ok": false, "msg": "authInvalidToken", "msgi18n": true });
    }

    let jwt_secret = state.jwt_secret.read().await;
    let token_data = match decode::<user::JwtClaims>(
        token,
        &DecodingKey::from_secret(jwt_secret.as_bytes()),
        &Validation::default(),
    ) {
        Ok(data) => data,
        Err(e) => {
            error!("Invalid token: {}", e);
            return json!({ "ok": false, "msg": "authInvalidToken", "msgi18n": true });
        }
    };

    let claims = token_data.claims;
    let user = match User::find_by_username(&state.db, &claims.username).await {
        Ok(Some(u)) => u,
        Ok(None) => {
            return json!({ "ok": false, "msg": "authUserInactiveOrDeleted", "msgi18n": true });
        }
        Err(e) => return error_response(&format!("{}", e)),
    };

    // Verify password hash hasn't changed
    let password_hash = user.password.as_deref().unwrap_or("");
    let h = user::shake256_hex(password_hash, 16);
    if h != claims.h {
        return json!({ "ok": false, "msg": "authInvalidToken", "msgi18n": true });
    }

    // Store user ID in socket extensions
    socket.extensions.insert(user.id);
    socket.extensions.insert(user.username.clone());

    info!("Successfully logged in user {} via token", claims.username);

    after_login(state, socket).await;

    json!({ "ok": true })
}

async fn handle_change_password(state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
    let user_id: Option<i64> = socket.extensions.get::<i64>();
    let user_id = match user_id {
        Some(id) => id,
        None => return error_response("Not logged in"),
    };

    let password_data = if data.is_array() {
        data.get(0).unwrap_or(data)
    } else {
        data
    };

    let current_password = password_data.get("currentPassword").and_then(|v| v.as_str()).unwrap_or("");
    let new_password = password_data.get("newPassword").and_then(|v| v.as_str()).unwrap_or("");

    if new_password.is_empty() {
        return error_response("Invalid new password");
    }

    if !check_password_strength(new_password) {
        return error_response(
            "Password is too weak. It should contain alphabetic and numeric characters. It must be at least 6 characters in length."
        );
    }

    // Verify current password
    let user = match User::find_by_id(&state.db, user_id).await {
        Ok(Some(u)) => u,
        _ => return error_response("User not found"),
    };

    if !user.verify_password(current_password) {
        return error_response("Incorrect current password");
    }

    let new_hash = generate_password_hash(new_password);
    if let Err(e) = User::update_password(&state.db, user_id, &new_hash).await {
        return error_response(&format!("{}", e));
    }

    json!({ "ok": true, "msg": "Password has been updated successfully." })
}

/// Actions to perform after successful login
pub async fn after_login(state: &Arc<AppState>, socket: &SocketRef) {
    // Send info
    send_info(state, socket, false).await;

    // Send stack list to this socket
    send_stack_list_to_socket(state, socket).await;

    // Send agent list (just local)
    send_agent_list(state, socket).await;
}

/// Send server info to a socket
pub async fn send_info(state: &Arc<AppState>, socket: &SocketRef, hide_version: bool) {
    let primary_hostname = settings::get(&state.db, "primaryHostname")
        .await
        .ok()
        .flatten()
        .and_then(|v| v.as_str().map(|s| s.to_string()));

    let mut info = json!({
        "primaryHostname": primary_hostname,
    });

    if !hide_version {
        let latest_version = state.latest_version.read().await;
        info["version"] = json!(state.version);
        info["latestVersion"] = json!(latest_version.as_deref());
        info["isContainer"] = json!(std::env::var("DOCKGE_IS_CONTAINER").ok() == Some("1".to_string()));
    }

    socket.emit("info", &info).ok();
}

/// Send the stack list to a single socket
pub async fn send_stack_list_to_socket(state: &Arc<AppState>, socket: &SocketRef) {
    let endpoint = socket.extensions.get::<String>()
        .map(|s| s.clone())
        .unwrap_or_default();

    let stack_cache = state.stack_cache.read().await;
    if stack_cache.is_empty() {
        // Refresh cache first
        drop(stack_cache);
        crate::handlers::stack::refresh_stack_cache(state).await;
        let stack_cache = state.stack_cache.read().await;
        let stack_list: serde_json::Map<String, Value> = stack_cache.iter()
            .map(|(k, v)| (k.clone(), serde_json::to_value(v).unwrap_or_default()))
            .collect();

        let data = json!({
            "ok": true,
            "stackList": stack_list,
            "endpoint": endpoint,
        });
        socket.emit("agent", &("stackList", data)).ok();
    } else {
        let stack_list: serde_json::Map<String, Value> = stack_cache.iter()
            .map(|(k, v)| (k.clone(), serde_json::to_value(v).unwrap_or_default()))
            .collect();

        let data = json!({
            "ok": true,
            "stackList": stack_list,
            "endpoint": endpoint,
        });
        socket.emit("agent", &("stackList", data)).ok();
    }
}

/// Send the agent list (stub — local only)
pub async fn send_agent_list(_state: &Arc<AppState>, socket: &SocketRef) {
    let data = json!({
        "ok": true,
        "agentList": {},
    });
    socket.emit("agent", &("agentList", data)).ok();
}

/// Check if a socket is logged in
pub fn check_login(socket: &SocketRef) -> Option<i64> {
    socket.extensions.get::<i64>()
}
