use std::collections::BTreeMap;
use std::sync::Arc;
use std::sync::atomic::Ordering;

use serde::Deserialize;
use tracing::{error, info, warn};

use crate::auth;
use crate::db::users;
use crate::docker;
use crate::ws::conn::Conn;
use crate::ws::protocol::{ClientMessage, ErrorResponse, ItemsEvent, OkResponse};
use crate::ws::WsServer;

use serde::Serialize;

use super::{parse_args, arg_string, arg_object, AppState};

pub fn register(ws: &mut WsServer, state: Arc<AppState>) {
    // login
    ws.handle_with_state("login", state.clone(), |state, conn, msg| async move {
        handle_login(&state, &conn, &msg).await;
    });

    // loginByToken
    ws.handle_with_state("loginByToken", state.clone(), |state, conn, msg| async move {
        handle_login_by_token(&state, &conn, &msg);
    });

    // logout (stateless — no state needed)
    ws.handle("logout", move |conn: Arc<Conn>, msg: ClientMessage| {
        async move {
            conn.set_user(0);
            if let Some(id) = msg.id {
                conn.send_ack(id, OkResponse::simple());
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
        #[derive(Serialize)]
        #[serde(rename_all = "camelCase")]
        struct NeedSetupResponse { ok: bool, need_setup: bool }

        if let Some(id) = msg.id {
            conn.send_ack(id, NeedSetupResponse {
                ok: true,
                need_setup: state.need_setup.load(Ordering::Relaxed),
            });
        }
    });

    // getTurnstileSiteKey (stateless stub)
    ws.handle("getTurnstileSiteKey", move |conn: Arc<Conn>, msg: ClientMessage| {
        async move {
            if let Some(id) = msg.id {
                conn.send_ack(id, OkResponse::simple());
            }
        }
    });

    // twoFAStatus (stateless stub)
    ws.handle("twoFAStatus", move |conn: Arc<Conn>, msg: ClientMessage| {
        async move {
            #[derive(Serialize)]
            struct TwoFAStatusResponse { ok: bool, status: bool }

            if let Some(id) = msg.id {
                conn.send_ack(id, TwoFAStatusResponse { ok: true, status: false });
            }
        }
    });

    // 2FA stubs
    for event in &["prepare2FA", "save2FA", "disable2FA", "verifyToken"] {
        ws.handle(event, move |conn: Arc<Conn>, msg: ClientMessage| {
            async move {
                if let Some(id) = msg.id {
                    conn.send_ack(id, ErrorResponse::new("2FA is not yet supported"));
                }
            }
        });
    }
}

async fn handle_login(state: &Arc<AppState>, conn: &Arc<Conn>, msg: &ClientMessage) {
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
            conn.send_ack(id, ErrorResponse::i18n("authIncorrectCreds"));
        }
        return;
    }

    // Rate limit
    if !state.login_limiter.allow(&username) {
        warn!(username = %username, "login rate limited");
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::new("Too many login attempts. Please try again later."));
        }
        return;
    }

    let user = match state.users.find_by_username(&username) {
        Ok(Some(u)) => u,
        Ok(None) => {
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::i18n("authIncorrectCreds"));
            }
            return;
        }
        Err(e) => {
            error!("login lookup: {e}");
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::new("Internal error"));
            }
            return;
        }
    };

    if !users::verify_password_async(&password, &user.password).await {
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::i18n("authIncorrectCreds"));
        }
        return;
    }

    let token = match auth::create_jwt(&user, &state.jwt_secret) {
        Ok(t) => t,
        Err(e) => {
            error!("create jwt: {e}");
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::new("Internal error"));
            }
            return;
        }
    };

    // If already authenticated (e.g. no-auth mode auto-auth'd on connect),
    // skip after_login — data was already sent. Still update user_id and ack.
    let was_authenticated = conn.is_authenticated();
    conn.set_user(user.id);
    state.has_authenticated.store(true, Ordering::Relaxed);
    state.login_limiter.reset(&username);

    if let Some(id) = msg.id {
        conn.send_ack(id, OkResponse {
            ok: true,
            msg: None,
            token: Some(token),
        });
    }

    info!(username = %username, "user logged in");

    if !was_authenticated {
        after_login(state, conn);
    }
}

fn handle_login_by_token(state: &Arc<AppState>, conn: &Arc<Conn>, msg: &ClientMessage) {
    let args = parse_args(msg);
    let token = arg_string(&args, 0);

    if token.is_empty() {
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::i18n("authInvalidToken"));
        }
        return;
    }

    let claims = match auth::verify_jwt(&token, &state.jwt_secret) {
        Ok(c) => c,
        Err(_) => {
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::i18n("authInvalidToken"));
            }
            return;
        }
    };

    let user = match state.users.find_by_username(&claims.username) {
        Ok(Some(u)) => u,
        Ok(None) => {
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::i18n("authUserInactiveOrDeleted"));
            }
            return;
        }
        Err(e) => {
            error!("token user lookup: {e}");
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::new("Internal error"));
            }
            return;
        }
    };

    // Password change detection
    if claims.h != auth::shake256_hex(&user.password) {
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::i18n("authInvalidToken"));
        }
        return;
    }

    let was_authenticated = conn.is_authenticated();
    conn.set_user(user.id);
    state.has_authenticated.store(true, Ordering::Relaxed);

    if let Some(id) = msg.id {
        conn.send_ack(id, OkResponse::simple());
    }

    if !was_authenticated {
        after_login(state, conn);
    }
}

async fn handle_setup(state: &AppState, conn: &Conn, msg: &ClientMessage) {
    let args = parse_args(msg);
    let username = arg_string(&args, 0);
    let password = arg_string(&args, 1);

    if username.is_empty() || password.is_empty() {
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::new("Username and password required"));
        }
        return;
    }

    if password.len() < 6 {
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::new("Password is too weak. It should be at least 6 characters."));
        }
        return;
    }

    let count = match state.users.count() {
        Ok(c) => c,
        Err(e) => {
            error!("setup count: {e}");
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::new("Internal error"));
            }
            return;
        }
    };

    if count > 0 {
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::new("Dockge has already been set up"));
        }
        return;
    }

    let hash = match users::hash_password_async(&password).await {
        Ok(h) => h,
        Err(e) => {
            error!("setup hash password: {e}");
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::new("Internal error"));
            }
            return;
        }
    };

    if let Err(e) = state.users.create_with_hash(&username, hash) {
        error!("setup create user: {e}");
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::new("Failed to create user"));
        }
        return;
    }

    state.need_setup.store(false, Ordering::Relaxed);

    #[derive(Serialize)]
    struct SetupSuccessResponse { ok: bool, msg: &'static str, msgi18n: bool }

    if let Some(id) = msg.id {
        conn.send_ack(id, SetupSuccessResponse {
            ok: true,
            msg: "successAdded",
            msgi18n: true,
        });
    }

    info!(username = %username, "setup complete");
}

async fn handle_change_password(state: &AppState, conn: &Conn, msg: &ClientMessage) {
    let uid = state.check_login(conn, msg);
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
                conn.send_ack(id, ErrorResponse::new("Invalid arguments"));
            }
            return;
        }
    };

    let user = match state.users.find_by_id(uid) {
        Ok(Some(u)) => u,
        _ => {
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::new("Internal error"));
            }
            return;
        }
    };

    if !users::verify_password_async(&data.current_password, &user.password).await {
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::i18n("authIncorrectCreds"));
        }
        return;
    }

    if data.new_password.len() < 6 {
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::new("Password too weak"));
        }
        return;
    }

    let hash = match users::hash_password_async(&data.new_password).await {
        Ok(h) => h,
        Err(e) => {
            error!("hash password: {e}");
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::new("Internal error"));
            }
            return;
        }
    };

    if let Err(e) = state.users.change_password_with_hash(uid, hash) {
        error!("change password: {e}");
        if let Some(id) = msg.id {
            conn.send_ack(id, ErrorResponse::new("Failed to change password"));
        }
        return;
    }

    // Broadcast "refresh" to all connections — forces re-login (session invalidated
    // because the password hash changed, so existing JWT tokens won't validate).
    state.broadcaster.send_event("refresh", &serde_json::Value::Null);

    if let Some(id) = msg.id {
        conn.send_ack(id, OkResponse {
            ok: true,
            msg: Some("Password changed".into()),
            token: None,
        });
    }
}

/// Send initial data to a freshly authenticated connection.
/// Each data source is spawned independently — results trickle in via the
/// connection's direct channel. Returns immediately.
pub(crate) fn after_login(state: &Arc<AppState>, conn: &Arc<Conn>) {
    // Stacks (sync filesystem I/O — use spawn_blocking)
    {
        let stacks_dir = state.config.stacks_dir.clone();
        let conn = conn.clone();
        tokio::task::spawn_blocking(move || {
            let stacks = build_stacks_broadcast(&stacks_dir);
            conn.send_event("stacks", ItemsEvent { items: stacks });
        });
    }

    // Containers
    {
        let docker = state.docker.clone();
        let conn = conn.clone();
        tokio::spawn(async move {
            match docker::container_list(&docker, None).await {
                Ok(containers) => {
                    let map = containers_to_map(containers);
                    conn.send_event("containers", ItemsEvent { items: map });
                }
                Err(e) => warn!("afterLogin: containers: {e}"),
            }
        });
    }

    // Networks
    {
        let docker = state.docker.clone();
        let conn = conn.clone();
        tokio::spawn(async move {
            match docker::network_list(&docker).await {
                Ok(networks) => {
                    let map: BTreeMap<String, _> = networks.into_iter().map(|n| (n.name.clone(), n)).collect();
                    conn.send_event("networks", ItemsEvent { items: map });
                }
                Err(e) => warn!("afterLogin: networks: {e}"),
            }
        });
    }

    // Images
    {
        let docker = state.docker.clone();
        let conn = conn.clone();
        tokio::spawn(async move {
            match docker::image_list(&docker).await {
                Ok(images) => {
                    let map: BTreeMap<String, _> = images.into_iter().map(|i| (i.id.clone(), i)).collect();
                    conn.send_event("images", ItemsEvent { items: map });
                }
                Err(e) => warn!("afterLogin: images: {e}"),
            }
        });
    }

    // Volumes
    {
        let docker = state.docker.clone();
        let conn = conn.clone();
        tokio::spawn(async move {
            match docker::volume_list(&docker).await {
                Ok(volumes) => {
                    let map: BTreeMap<String, _> = volumes.into_iter().map(|v| (v.name.clone(), v)).collect();
                    conn.send_event("volumes", ItemsEvent { items: map });
                }
                Err(e) => warn!("afterLogin: volumes: {e}"),
            }
        });
    }

    // Image updates (sync redb I/O — use spawn_blocking)
    {
        let state = state.clone();
        let conn = conn.clone();
        tokio::task::spawn_blocking(move || {
            let updates = super::image_updates::collect_update_keys(&state);
            conn.send_event("updates", updates);
        });
    }
}

/// Stack metadata for the "stacks" broadcast event.
#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub(crate) struct StackBroadcast {
    name: String,
    compose_file_name: String,
    is_managed_by_dockge: bool,
    images: BTreeMap<String, String>,
    #[serde(skip_serializing_if = "BTreeMap::is_empty")]
    ignore_status: BTreeMap<String, bool>,
}

pub(crate) fn build_stacks_broadcast(stacks_dir: &str) -> BTreeMap<String, StackBroadcast> {
    let mut result = BTreeMap::new();
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

        // Parse compose file to extract service→image mappings and labels
        let compose_path = path.join(compose_file);
        let yaml_content = std::fs::read_to_string(&compose_path).unwrap_or_default();
        let images: BTreeMap<String, String> = super::image_updates::parse_service_images(&yaml_content)
            .into_iter()
            .collect();
        let ignore_status: BTreeMap<String, bool> = parse_status_ignore_labels(&yaml_content)
            .into_iter()
            .collect();

        result.insert(name.clone(), StackBroadcast {
            name: name.clone(),
            compose_file_name: compose_file.to_string(),
            is_managed_by_dockge: true,
            images,
            ignore_status,
        });
    }

    result
}

/// Parse `dockge.status.ignore` labels from compose YAML.
/// Returns a vec of (service_name, true) for services that have the label set.
fn parse_status_ignore_labels(yaml: &str) -> Vec<(String, bool)> {
    let mut results = Vec::new();
    let mut in_services = false;
    let mut current_service: Option<String> = None;
    let mut in_labels = false;

    for line in yaml.lines() {
        let trimmed = line.trim();
        if trimmed.is_empty() || trimmed.starts_with('#') {
            continue;
        }

        let indent = line.len() - line.trim_start().len();

        // Top-level key
        if indent == 0 {
            in_services = trimmed == "services:" || trimmed.starts_with("services:");
            current_service = None;
            in_labels = false;
            continue;
        }

        if !in_services {
            continue;
        }

        // Service name (indent 2, ends with ':')
        if indent <= 2 && trimmed.ends_with(':') && !trimmed.contains(' ') {
            current_service = Some(trimmed.trim_end_matches(':').to_string());
            in_labels = false;
            continue;
        }

        if current_service.is_none() {
            continue;
        }

        // Service property level (indent 4-6)
        if (4..=6).contains(&indent) && trimmed == "labels:" {
            in_labels = true;
            continue;
        }

        // Exiting labels section (back to service property level)
        if (4..=6).contains(&indent) && trimmed.ends_with(':') && trimmed != "labels:" {
            in_labels = false;
            continue;
        }

        // Label value (indent 6+)
        if in_labels && indent >= 6
            && let Some(val) = trimmed.strip_prefix("dockge.status.ignore:")
        {
            let val = val.trim().trim_matches('"').trim_matches('\'');
            if val == "true" {
                results.push((current_service.clone().unwrap(), true));
            }
        }
    }

    results
}

/// Build a name → container map. Values are `Option` so that destroyed
/// containers can be represented as `None` (serialized as JSON `null`).
pub fn containers_to_map(containers: Vec<docker::types::ContainerBroadcast>) -> BTreeMap<String, Option<docker::types::ContainerBroadcast>> {
    containers.into_iter().map(|c| (c.name.clone(), Some(c))).collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_status_ignore_labels() {
        let yaml = r#"services:
  nginx:
    image: nginx:latest
    restart: unless-stopped
    ports:
      - "8080:80"
    labels:
      dockge.imageupdates.changelog: "https://github.com/nginx/nginx/releases"
      dockge.urls.web: "http://localhost:8080"
  redis:
    image: redis:alpine
    restart: unless-stopped
    labels:
      dockge.status.ignore: "true"
"#;
        let result = parse_status_ignore_labels(yaml);
        assert_eq!(result, vec![("redis".to_string(), true)]);
    }

    #[test]
    fn test_stack_broadcast_serializes_ignore_status() {
        let mut ignore = BTreeMap::new();
        ignore.insert("redis".to_string(), true);
        let sb = StackBroadcast {
            name: "test".to_string(),
            compose_file_name: "compose.yaml".to_string(),
            is_managed_by_dockge: true,
            images: BTreeMap::new(),
            ignore_status: ignore,
        };
        let json = serde_json::to_string(&sb).unwrap();
        assert!(json.contains("\"ignoreStatus\""), "Expected ignoreStatus in JSON: {json}");
        assert!(json.contains("\"redis\":true"), "Expected redis:true in JSON: {json}");
    }

    #[test]
    fn test_parse_no_labels_section() {
        let yaml = "services:\n  web:\n    image: nginx:latest\n";
        let result = parse_status_ignore_labels(yaml);
        assert!(result.is_empty());
    }

    #[test]
    fn test_parse_ignore_false_value() {
        let yaml = "services:\n  web:\n    image: nginx\n    labels:\n      dockge.status.ignore: \"false\"\n";
        let result = parse_status_ignore_labels(yaml);
        assert!(result.is_empty());
    }

    #[test]
    fn test_parse_multiple_services_with_ignore() {
        let yaml = r#"services:
  web:
    image: nginx
    labels:
      dockge.status.ignore: "true"
  db:
    image: postgres
    labels:
      dockge.status.ignore: "true"
"#;
        let result = parse_status_ignore_labels(yaml);
        assert_eq!(result.len(), 2);
        assert!(result.contains(&("web".to_string(), true)));
        assert!(result.contains(&("db".to_string(), true)));
    }

    #[test]
    fn test_parse_empty_yaml() {
        assert!(parse_status_ignore_labels("").is_empty());
    }

    #[test]
    fn test_stack_broadcast_omits_empty_ignore_status() {
        let sb = StackBroadcast {
            name: "test".to_string(),
            compose_file_name: "compose.yaml".to_string(),
            is_managed_by_dockge: true,
            images: BTreeMap::new(),
            ignore_status: BTreeMap::new(),
        };
        let json = serde_json::to_string(&sb).unwrap();
        assert!(!json.contains("ignoreStatus"), "Should omit empty ignoreStatus: {json}");
    }
}
