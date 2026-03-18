use std::path::Path;
use std::sync::Arc;

use serde::Deserialize;
use tracing::{error, info, warn};

use crate::ws::conn::Conn;
use crate::ws::protocol::{ClientMessage, ErrorResponse, OkResponse};
use crate::ws::WsServer;

use super::{arg_object, arg_string, parse_args, AppState};

/// Validate stack name: reject path traversal, shell injection, uppercase, dots, spaces, etc.
fn validate_stack_name(name: &str) -> Result<(), &'static str> {
    if name.is_empty() {
        return Err("Stack name required");
    }
    if name.contains("..") || name.contains('/') || name.contains('\\') {
        return Err("Invalid stack name (path traversal)");
    }
    if name.contains('\0') {
        return Err("Invalid stack name (null byte)");
    }
    if name.contains(';')
        || name.contains('|')
        || name.contains('&')
        || name.contains('`')
        || name.contains('$')
        || name.contains('>')
        || name.contains('<')
        || name.contains('(')
        || name.contains(')')
    {
        return Err("Invalid stack name (shell characters)");
    }
    if name.starts_with('.') {
        return Err("Invalid stack name (dot prefix)");
    }
    if name.starts_with('-') {
        return Err("Invalid stack name (leading hyphen)");
    }
    if name.contains(' ') {
        return Err("Invalid stack name (spaces)");
    }
    if name.chars().any(|c| c.is_uppercase()) {
        return Err("Invalid stack name (uppercase)");
    }
    Ok(())
}

/// Per-stack named mutex for write serialization.
pub struct NamedMutex {
    locks: std::sync::Mutex<std::collections::HashMap<String, Arc<tokio::sync::Mutex<()>>>>,
}

impl NamedMutex {
    pub fn new() -> Self {
        Self {
            locks: std::sync::Mutex::new(std::collections::HashMap::new()),
        }
    }

    pub fn get(&self, name: &str) -> Arc<tokio::sync::Mutex<()>> {
        let mut locks = self.locks.lock().unwrap();
        locks
            .entry(name.to_string())
            .or_insert_with(|| Arc::new(tokio::sync::Mutex::new(())))
            .clone()
    }
}

/// Find the compose file for a stack (checks multiple filenames).
fn find_compose_file(stacks_dir: &str, stack_name: &str) -> Option<String> {
    for filename in &[
        "compose.yaml",
        "docker-compose.yml",
        "docker-compose.yaml",
        "compose.yml",
    ] {
        let path = format!("{}/{}/{}", stacks_dir, stack_name, filename);
        if Path::new(&path).exists() {
            return Some(filename.to_string());
        }
    }
    None
}

/// Check if a stack is managed (has a compose file on disk).
#[allow(dead_code)]
fn is_stack_managed(stacks_dir: &str, stack_name: &str) -> bool {
    find_compose_file(stacks_dir, stack_name).is_some()
}

pub fn register(ws: &mut WsServer, state: Arc<AppState>) {
    // Register stack lifecycle handlers (loop — keep on ws.handle)
    for (event, action) in &[
        ("startStack", "start"),
        ("stopStack", "stop"),
        ("restartStack", "restart"),
        ("downStack", "down"),
        ("pauseStack", "pause"),
        ("resumeStack", "unpause"),
    ] {
        let state = state.clone();
        let action = action.to_string();
        let event_name = event.to_string();
        ws.handle(event, move |conn: Arc<Conn>, msg: ClientMessage| {
            let state = state.clone();
            let action = action.clone();
            let event_name = event_name.clone();
            async move {
                let uid = state.check_login(&conn, &msg).await;
                if uid == 0 {
                    return;
                }

                let args = parse_args(&msg);
                let stack_name = arg_string(&args, 0);

                if let Err(e) = validate_stack_name(&stack_name) {
                    if let Some(id) = msg.id {
                        conn.send_ack(id, ErrorResponse::new(e)).await;
                    }
                    return;
                }

                // Respond immediately
                if let Some(id) = msg.id {
                    conn.send_ack(
                        id,
                        OkResponse {
                            ok: true,
                            msg: None,
                            token: None,
                        },
                    )
                    .await;
                }

                // Run compose action in background with per-stack lock
                let lock = state.stack_locks.get(&stack_name);
                let _guard = lock.lock().await;

                let stacks_dir = state.config.stacks_dir.clone();
                let stack_dir = format!("{}/{}", stacks_dir, stack_name);

                let result = run_compose_action(&stack_dir, &action).await;
                match &result {
                    Ok(_) => {
                        info!(stack = %stack_name, action = %action, "compose action completed")
                    }
                    Err(e) => {
                        warn!(stack = %stack_name, action = %action, "compose action failed: {e}")
                    }
                }
                let _ = (&event_name,);
            }
        });
    }

    // getStack
    ws.handle_with_state("getStack", state.clone(), |state, conn, msg| async move {
        let uid = state.check_login(&conn, &msg).await;
        if uid == 0 {
            return;
        }

        let args = parse_args(&msg);
        let stack_name = arg_string(&args, 0);

        if let Err(e) = validate_stack_name(&stack_name) {
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::new(e)).await;
            }
            return;
        }

        let stacks_dir = &state.config.stacks_dir;
        let compose_path =
            format!("{}/{}/compose.yaml", stacks_dir, stack_name);
        let compose_yaml =
            std::fs::read_to_string(&compose_path).unwrap_or_default();
        let env_path = format!("{}/{}/.env", stacks_dir, stack_name);
        let compose_env =
            std::fs::read_to_string(&env_path).unwrap_or_default();

        if let Some(id) = msg.id {
            conn.send_ack(
                id,
                serde_json::json!({
                    "ok": true,
                    "stack": {
                        "name": stack_name,
                        "composeYAML": compose_yaml,
                        "composeENV": compose_env,
                        "isManagedByDockge": true,
                    }
                }),
            )
            .await;
        }
    });

    // saveStack
    ws.handle_with_state("saveStack", state.clone(), |state, conn, msg| async move {
        let uid = state.check_login(&conn, &msg).await;
        if uid == 0 {
            return;
        }
        let args = parse_args(&msg);
        let stack_name = arg_string(&args, 0);
        let compose_yaml = arg_string(&args, 1);
        let compose_env = arg_string(&args, 2);

        if stack_name.is_empty() || compose_yaml.is_empty() {
            if let Some(id) = msg.id {
                conn.send_ack(
                    id,
                    ErrorResponse::new(
                        "Stack name and compose YAML required",
                    ),
                )
                .await;
            }
            return;
        }
        if let Err(e) = validate_stack_name(&stack_name) {
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::new(e)).await;
            }
            return;
        }

        let lock = state.stack_locks.get(&stack_name);
        let _guard = lock.lock().await;

        let stack_dir = format!(
            "{}/{}",
            state.config.stacks_dir, stack_name
        );
        if let Err(e) = save_stack_to_disk(
            &stack_dir,
            &compose_yaml,
            &compose_env,
        ) {
            error!(stack = %stack_name, "save stack: {e}");
            if let Some(id) = msg.id {
                conn.send_ack(
                    id,
                    ErrorResponse::new(e.to_string()),
                )
                .await;
            }
            return;
        }

        if let Some(id) = msg.id {
            conn.send_ack(
                id,
                OkResponse {
                    ok: true,
                    msg: Some("Saved".into()),
                    token: None,
                },
            )
            .await;
        }
        info!(stack = %stack_name, "stack saved");
    });

    // deployStack
    ws.handle_with_state("deployStack", state.clone(), |state, conn, msg| async move {
        let uid = state.check_login(&conn, &msg).await;
        if uid == 0 {
            return;
        }
        let args = parse_args(&msg);
        let stack_name = arg_string(&args, 0);
        let compose_yaml = arg_string(&args, 1);
        let compose_env = arg_string(&args, 2);

        if stack_name.is_empty() || compose_yaml.is_empty() {
            if let Some(id) = msg.id {
                conn.send_ack(
                    id,
                    ErrorResponse::new(
                        "Stack name and compose YAML required",
                    ),
                )
                .await;
            }
            return;
        }
        if let Err(e) = validate_stack_name(&stack_name) {
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::new(e)).await;
            }
            return;
        }

        let lock = state.stack_locks.get(&stack_name);
        let _guard = lock.lock().await;

        let stacks_dir = state.config.stacks_dir.clone();
        let stack_dir =
            format!("{}/{}", stacks_dir, stack_name);

        // Save compose files to disk
        if let Err(e) = save_stack_to_disk(
            &stack_dir,
            &compose_yaml,
            &compose_env,
        ) {
            error!(stack = %stack_name, "deploy save: {e}");
            if let Some(id) = msg.id {
                conn.send_ack(
                    id,
                    ErrorResponse::new(e.to_string()),
                )
                .await;
            }
            return;
        }

        // Deploy: docker compose up -d --remove-orphans
        let result = run_compose_action_with_args(
            &stack_dir,
            &["up", "-d", "--remove-orphans"],
        )
        .await;
        match &result {
            Ok(_) => info!(stack = %stack_name, "deploy completed"),
            Err(e) => {
                warn!(stack = %stack_name, "deploy failed: {e}")
            }
        }

        if let Some(id) = msg.id {
            conn.send_ack(
                id,
                OkResponse {
                    ok: true,
                    msg: Some("Deployed".into()),
                    token: None,
                },
            )
            .await;
        }
    });

    // deleteStack
    ws.handle_with_state("deleteStack", state.clone(), |state, conn, msg| async move {
        let uid = state.check_login(&conn, &msg).await;
        if uid == 0 {
            return;
        }
        let args = parse_args(&msg);
        let stack_name = arg_string(&args, 0);

        #[derive(Deserialize, Default)]
        #[serde(rename_all = "camelCase")]
        struct DeleteOpts {
            #[serde(default)]
            delete_stack_files: bool,
        }
        let opts: DeleteOpts =
            arg_object(&args, 1).unwrap_or_default();

        if let Err(e) = validate_stack_name(&stack_name) {
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::new(e)).await;
            }
            return;
        }

        // Ack immediately
        if let Some(id) = msg.id {
            conn.send_ack(
                id,
                OkResponse {
                    ok: true,
                    msg: None,
                    token: None,
                },
            )
            .await;
        }

        // Background: down + optionally delete files
        let stacks_dir = state.config.stacks_dir.clone();
        let stack_dir =
            format!("{}/{}", stacks_dir, stack_name);
        let lock = state.stack_locks.get(&stack_name);
        let _guard = lock.lock().await;

        let _ = run_compose_action_with_args(
            &stack_dir,
            &["down", "--remove-orphans"],
        )
        .await;

        if opts.delete_stack_files
            && let Err(e) = std::fs::remove_dir_all(&stack_dir)
        {
            error!(stack = %stack_name, "delete stack files: {e}");
        }
        info!(stack = %stack_name, "stack deleted");
    });

    // forceDeleteStack
    ws.handle_with_state("forceDeleteStack", state.clone(), |state, conn, msg| async move {
        let uid = state.check_login(&conn, &msg).await;
        if uid == 0 {
            return;
        }
        let args = parse_args(&msg);
        let stack_name = arg_string(&args, 0);
        if let Err(e) = validate_stack_name(&stack_name) {
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::new(e)).await;
            }
            return;
        }

        // Ack immediately
        if let Some(id) = msg.id {
            conn.send_ack(
                id,
                OkResponse {
                    ok: true,
                    msg: None,
                    token: None,
                },
            )
            .await;
        }

        // Background: down -v + delete files
        let stacks_dir = state.config.stacks_dir.clone();
        let stack_dir =
            format!("{}/{}", stacks_dir, stack_name);
        let lock = state.stack_locks.get(&stack_name);
        let _guard = lock.lock().await;

        let _ = run_compose_action_with_args(
            &stack_dir,
            &["down", "-v", "--remove-orphans"],
        )
        .await;

        if let Err(e) = std::fs::remove_dir_all(&stack_dir) {
            error!(stack = %stack_name, "force delete stack: {e}");
        }
        info!(stack = %stack_name, "stack force deleted");
    });

    // updateStack
    ws.handle_with_state("updateStack", state.clone(), |state, conn, msg| async move {
        let uid = state.check_login(&conn, &msg).await;
        if uid == 0 {
            return;
        }
        let args = parse_args(&msg);
        let stack_name = arg_string(&args, 0);
        if let Err(e) = validate_stack_name(&stack_name) {
            if let Some(id) = msg.id {
                conn.send_ack(id, ErrorResponse::new(e)).await;
            }
            return;
        }

        // Ack immediately
        if let Some(id) = msg.id {
            conn.send_ack(
                id,
                OkResponse {
                    ok: true,
                    msg: None,
                    token: None,
                },
            )
            .await;
        }

        // Background: pull + up
        let stacks_dir = state.config.stacks_dir.clone();
        let stack_dir =
            format!("{}/{}", stacks_dir, stack_name);
        let lock = state.stack_locks.get(&stack_name);
        let _guard = lock.lock().await;

        let _ = run_compose_action_with_args(
            &stack_dir,
            &["pull"],
        )
        .await;
        let _ = run_compose_action_with_args(
            &stack_dir,
            &["up", "-d", "--remove-orphans"],
        )
        .await;

        info!(stack = %stack_name, "stack updated");
    });
}

/// Save compose YAML and .env to disk.
fn save_stack_to_disk(
    stack_dir: &str,
    compose_yaml: &str,
    compose_env: &str,
) -> Result<(), std::io::Error> {
    std::fs::create_dir_all(stack_dir)?;
    std::fs::write(
        format!("{}/compose.yaml", stack_dir),
        compose_yaml,
    )?;
    let env_path = format!("{}/.env", stack_dir);
    if compose_env.is_empty() {
        // Remove .env if empty
        let _ = std::fs::remove_file(&env_path);
    } else {
        std::fs::write(&env_path, compose_env)?;
    }
    Ok(())
}

/// Run a docker compose action in a stack directory.
async fn run_compose_action(
    stack_dir: &str,
    action: &str,
) -> Result<String, Box<dyn std::error::Error + Send + Sync>> {
    run_compose_action_with_args(stack_dir, &[action]).await
}

/// Run a docker compose action with multiple args in a stack directory.
async fn run_compose_action_with_args(
    stack_dir: &str,
    args: &[&str],
) -> Result<String, Box<dyn std::error::Error + Send + Sync>> {
    let mut cmd_args = vec!["compose"];
    cmd_args.extend_from_slice(args);

    let output = tokio::process::Command::new("docker")
        .args(&cmd_args)
        .current_dir(stack_dir)
        .output()
        .await?;

    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        let args_str = cmd_args.join(" ");
        return Err(format!("docker {args_str} failed: {stderr}").into());
    }

    Ok(String::from_utf8_lossy(&output.stdout).to_string())
}
