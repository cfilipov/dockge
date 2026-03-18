use std::collections::HashMap;
use std::sync::Arc;
use std::sync::atomic::Ordering;

use serde::Deserialize;
use tracing::{error, info, warn};

use crate::auth;
use crate::db::users;
use crate::docker;
use crate::ws::conn::Conn;
use crate::ws::protocol::{ClientMessage, ErrorResponse, OkResponse};
use crate::ws::WsServer;

use super::{parse_args, arg_string, arg_object, AppState};

pub fn register(ws: &mut WsServer, state: Arc<AppState>) {
    // login
    ws.handle_with_state("login", state.clone(), |state, conn, msg| async move {
        handle_login(&state, &conn, &msg).await;
    });

    // loginByToken
    ws.handle_with_state("loginByToken", state.clone(), |state, conn, msg| async move {
        handle_login_by_token(&state, &conn, &msg).await;
    });

    // logout (stateless — no state needed)
    ws.handle("logout", move |conn: Arc<Conn>, msg: ClientMessage| {
        async move {
            conn.set_user(0);
            if let Some(id) = msg.id {
                conn.send_ack(id, OkResponse { ok: true, msg: None, token: None }).await;
            }
        }
    });

    // setup
    ws.handle_with_state("setup", state.clone(), |state, conn, msg| async move {
        handle_setup(&state, &conn, &msg).await;
    });

    // changePassword
    ws.handle_with_state("changePassword", state.clone(), |state, conn, msg| async move {
        handle_change_password(&state, &conn, &msg).await;
    });

    // needSetup
    ws.handle_with_state("needSetup", state.clone(), |state, conn, msg| async move {
        if let Some(id) = msg.id {
            conn.send_ack(id, serde_json::json!({
                "ok": true,
                "needSetup": state.need_setup.load(Ordering::Relaxed),
            })).await;
        }
    });

    // getTurnstileSiteKey (stateless stub)
    ws.handle("getTurnstileSiteKey", move |conn: Arc<Conn>, msg: ClientMessage| {
        async move {
            if let Some(id) = msg.id {
                conn.send_ack(id, OkResponse { ok: true, msg: None, token: None }).await;
            }
        }
    });

    // twoFAStatus (stateless stub)
    ws.handle("twoFAStatus", move |conn: Arc<Conn>, msg: ClientMessage| {
        async move {
            if let Some(id) = msg.id {
                conn.send_ack(id, serde_json::json!({
                    "ok": true,
                    "status": false,
                })).await;
            }
        }
    });

    // 2FA stubs
    for event in &["prepare2FA", "save2FA", "disable2FA", "verifyToken"] {
        ws.handle(event, move |conn: Arc<Conn>, msg: ClientMessage| {
            async move {
                if let Some(id) = msg.id {
                    conn.send_ack(id, ErrorResponse::new("2FA is not yet supported")).await;
                }
            }
        });
    }
}

async fn handle_login(state: &AppState, conn: &Conn, msg: &ClientMessage) {
    let args = parse_args(msg);

    // Try positional args: [username, password, token, captchaToken]
    let (username, password) = if !args.is_empty() {
        // Try object format first
        #[derive(Deserialize)]
        struct LoginData {
            #[serde(default)]
            username: String,
            #[serde(default)]
            password: String,
        }
        if let Some(data) = arg_object::<LoginData>(&args, 0) {
            if !data.username.is_empty() {
                (data.username, data.password)
            } else {
                (arg_string(&args, 0), arg_string(&args, 1))
            }
        } else {
            (arg_string(&args, 0), arg_string(&args, 1))
        }
    } else {
        (String::new(), String::new())
    };

    if username.is_empty() || password.is_empty() {
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::i18n("authIncorrectCreds")).await;
        }
        return;
    }

    // Rate limit
    if !state.login_limiter.allow(&username) {
        warn!(username = %username, "login rate limited");
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::new("Too many login attempts. Please try again later.")).await;
        }
        return;
    }

    let user = match state.users.find_by_username(&username) {
        Ok(Some(u)) => u,
        Ok(None) => {
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::i18n("authIncorrectCreds")).await;
            }
            return;
        }
        Err(e) => {
            error!("login lookup: {e}");
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::new("Internal error")).await;
            }
            return;
        }
    };

    if !users::verify_password(&password, &user.password) {
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::i18n("authIncorrectCreds")).await;
        }
        return;
    }

    let token = match auth::create_jwt(&user, &state.jwt_secret) {
        Ok(t) => t,
        Err(e) => {
            error!("create jwt: {e}");
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::new("Internal error")).await;
            }
            return;
        }
    };

    conn.set_user(user.id);
    state.has_authenticated.store(true, Ordering::Relaxed);
    state.login_limiter.reset(&username);

    if let Some(id) = msg.id {
        conn.send_ack(id, OkResponse {
            ok: true,
            msg: None,
            token: Some(token),
        }).await;
    }

    info!(username = %username, "user logged in");

    // AfterLogin: send initial data broadcasts (after ack so frontend is ready)
    after_login(state, conn).await;
}

async fn handle_login_by_token(state: &AppState, conn: &Conn, msg: &ClientMessage) {
    let args = parse_args(msg);
    let token = arg_string(&args, 0);

    if token.is_empty() {
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::i18n("authInvalidToken")).await;
        }
        return;
    }

    let claims = match auth::verify_jwt(&token, &state.jwt_secret) {
        Ok(c) => c,
        Err(_) => {
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::i18n("authInvalidToken")).await;
            }
            return;
        }
    };

    let user = match state.users.find_by_username(&claims.username) {
        Ok(Some(u)) => u,
        Ok(None) => {
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::i18n("authUserInactiveOrDeleted")).await;
            }
            return;
        }
        Err(e) => {
            error!("token user lookup: {e}");
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::new("Internal error")).await;
            }
            return;
        }
    };

    // Password change detection
    if claims.h != auth::shake256_hex(&user.password) {
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::i18n("authInvalidToken")).await;
        }
        return;
    }

    conn.set_user(user.id);
    state.has_authenticated.store(true, Ordering::Relaxed);

    if let Some(id) = msg.id {
        conn.send_ack(id, OkResponse { ok: true, msg: None, token: None }).await;
    }

    // AfterLogin: send initial data broadcasts (after ack so frontend is ready)
    after_login(state, conn).await;
}

async fn handle_setup(state: &AppState, conn: &Conn, msg: &ClientMessage) {
    let args = parse_args(msg);
    let username = arg_string(&args, 0);
    let password = arg_string(&args, 1);

    if username.is_empty() || password.is_empty() {
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::new("Username and password required")).await;
        }
        return;
    }

    if password.len() < 6 {
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::new("Password is too weak. It should be at least 6 characters.")).await;
        }
        return;
    }

    let count = match state.users.count() {
        Ok(c) => c,
        Err(e) => {
            error!("setup count: {e}");
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::new("Internal error")).await;
            }
            return;
        }
    };

    if count > 0 {
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::new("Dockge has already been set up")).await;
        }
        return;
    }

    if let Err(e) = state.users.create(&username, &password) {
        error!("setup create user: {e}");
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::new("Failed to create user")).await;
        }
        return;
    }

    state.need_setup.store(false, Ordering::Relaxed);

    if let Some(id) = msg.id {
        conn.send_ack(id, serde_json::json!({
            "ok": true,
            "msg": "successAdded",
            "msgi18n": true,
        })).await;
    }

    info!(username = %username, "setup complete");
}

async fn handle_change_password(state: &AppState, conn: &Conn, msg: &ClientMessage) {
    let uid = state.check_login(conn, msg).await;
    if uid == 0 {
        return;
    }

    let args = parse_args(msg);

    #[derive(Deserialize)]
    #[serde(rename_all = "camelCase")]
    struct ChangePasswordData {
        current_password: String,
        new_password: String,
    }

    let data: ChangePasswordData = match arg_object(&args, 0) {
        Some(d) => d,
        None => {
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::new("Invalid arguments")).await;
            }
            return;
        }
    };

    let user = match state.users.find_by_id(uid) {
        Ok(Some(u)) => u,
        _ => {
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::new("Internal error")).await;
            }
            return;
        }
    };

    if !users::verify_password(&data.current_password, &user.password) {
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::i18n("authIncorrectCreds")).await;
        }
        return;
    }

    if data.new_password.len() < 6 {
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::new("Password too weak")).await;
        }
        return;
    }

    if let Err(e) = state.users.change_password(uid, &data.new_password) {
        error!("change password: {e}");
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::new("Failed to change password")).await;
        }
        return;
    }

    if let Some(id) = msg.id {
        conn.send_ack(id, OkResponse {
            ok: true,
            msg: Some("Password changed".into()),
            token: None,
        }).await;
    }
}

/// Send initial data to a freshly authenticated connection.
/// Each broadcast fires independently — no channel waits on any other.
async fn after_login(state: &AppState, conn: &Conn) {
    // Stacks broadcast
    {
        let stacks_dir = state.config.stacks_dir.clone();
        let stacks = build_stacks_broadcast(&stacks_dir);
        conn.send_event("stacks", serde_json::json!({"items": stacks})).await;
    }

    // Containers broadcast
    {
        let docker = &state.docker;
        match docker::container_list(docker, None).await {
            Ok(containers) => {
                let map = containers_to_map(containers);
                conn.send_event("containers", serde_json::json!({"items": map})).await;
            }
            Err(e) => warn!("afterLogin: containers: {e}"),
        }
    }

    // Networks broadcast
    {
        match docker::network_list(&state.docker).await {
            Ok(networks) => {
                let map: HashMap<String, _> = networks.into_iter().map(|n| (n.name.clone(), n)).collect();
                conn.send_event("networks", serde_json::json!({"items": map})).await;
            }
            Err(e) => warn!("afterLogin: networks: {e}"),
        }
    }

    // Images broadcast
    {
        match docker::image_list(&state.docker).await {
            Ok(images) => {
                let map: HashMap<String, _> = images.into_iter().map(|i| (i.id.clone(), i)).collect();
                conn.send_event("images", serde_json::json!({"items": map})).await;
            }
            Err(e) => warn!("afterLogin: images: {e}"),
        }
    }

    // Volumes broadcast
    {
        match docker::volume_list(&state.docker).await {
            Ok(volumes) => {
                let map: HashMap<String, _> = volumes.into_iter().map(|v| (v.name.clone(), v)).collect();
                conn.send_event("volumes", serde_json::json!({"items": map})).await;
            }
            Err(e) => warn!("afterLogin: volumes: {e}"),
        }
    }

    // Updates broadcast (empty for now — image update checks are M8)
    // Sent as raw array, NOT wrapped in {"items": ...} — the frontend
    // expects a plain string[] for the updates channel.
    {
        let updates: Vec<String> = Vec::new();
        conn.send_event("updates", updates).await;
    }
}

pub(crate) fn build_stacks_broadcast(stacks_dir: &str) -> HashMap<String, serde_json::Value> {
    let mut result = HashMap::new();
    let dir = match std::fs::read_dir(stacks_dir) {
        Ok(d) => d,
        Err(_) => return result,
    };

    for entry in dir.flatten() {
        let name = entry.file_name().to_string_lossy().to_string();
        if name.starts_with('.') { continue; }
        let path = entry.path();
        if !path.is_dir() { continue; }

        // Check for compose file
        let compose_file = if path.join("compose.yaml").exists() {
            "compose.yaml"
        } else if path.join("docker-compose.yml").exists() {
            "docker-compose.yml"
        } else if path.join("docker-compose.yaml").exists() {
            "docker-compose.yaml"
        } else {
            continue;
        };

        result.insert(name.clone(), serde_json::json!({
            "name": name,
            "composeFileName": compose_file,
            "isManagedByDockge": true,
            "images": {},
        }));
    }

    result
}

pub fn containers_to_map(containers: Vec<docker::types::ContainerBroadcast>) -> HashMap<String, serde_json::Value> {
    let mut map = HashMap::new();
    for c in containers {
        let key = c.name.clone();
        map.insert(key, serde_json::to_value(&c).unwrap_or_default());
    }
    map
}
