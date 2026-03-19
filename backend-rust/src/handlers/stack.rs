use std::path::Path;
use std::sync::Arc;

use std::time::Duration;

use serde::{Deserialize, Serialize};
use tracing::{error, info, warn};

use crate::terminal::TerminalType;
use crate::ws::conn::Conn;
use crate::ws::protocol::{ClientMessage, ErrorResponse, OkResponse};
use crate::ws::WsServer;

use super::{arg_object, arg_string, parse_args, run_pty_to_terminal, AppState};

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
pub(crate) fn find_compose_file(stacks_dir: &str, stack_name: &str) -> Option<String> {
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
pub(crate) fn is_stack_managed(stacks_dir: &str, stack_name: &str) -> bool {
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
        ws.handle(event, move |conn: Arc<Conn>, msg: ClientMessage| {
            let state = state.clone();
            let action = action.clone();
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

                if let Some(id) = msg.id {
                    conn.send_ack(id, OkResponse::simple()).await;
                }

                // Run compose action in background so the worker is free
                // to process subsequent messages (e.g. terminalJoin).
                tokio::spawn(async move {
                    let lock = state.stack_locks.get(&stack_name);
                    let _guard = lock.lock().await;

                    let stack_dir = format!("{}/{}", state.config.stacks_dir, stack_name);
                    let term_name = format!("compose-{}", stack_name);
                    state.terminal_manager.recreate(&term_name, TerminalType::Pty).await;

                    let cmd_display = format!("$ docker compose {action}\r\n");
                    state.terminal_manager.write_data(&term_name, cmd_display.into_bytes());

                    match run_pty_to_terminal(
                        &state.terminal_manager, &term_name,
                        "docker", &["compose", &action], Some(&stack_dir),
                    ).await {
                        Ok(()) => info!(stack = %stack_name, action = %action, "compose action completed"),
                        Err(e) => warn!(stack = %stack_name, action = %action, "compose action failed: {e}"),
                    }
                    state.terminal_manager.remove_after(&term_name, Duration::from_secs(30));
                });
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
        let compose_file_name = find_compose_file(stacks_dir, &stack_name)
            .unwrap_or_else(|| "compose.yaml".to_string());
        let compose_path =
            format!("{}/{}/{}", stacks_dir, stack_name, compose_file_name);
        let compose_yaml =
            std::fs::read_to_string(&compose_path).unwrap_or_default();
        let env_path = format!("{}/{}/.env", stacks_dir, stack_name);
        let compose_env =
            std::fs::read_to_string(&env_path).unwrap_or_default();

        #[derive(Serialize)]
        struct GetStackResponse { ok: bool, stack: StackData }
        #[derive(Serialize)]
        struct StackData {
            name: String,
            #[serde(rename = "composeYAML")]
            compose_yaml: String,
            #[serde(rename = "composeENV")]
            compose_env: String,
            #[serde(rename = "composeFileName")]
            compose_file_name: String,
            #[serde(rename = "isManagedByDockge")]
            is_managed_by_dockge: bool,
        }

        if let Some(id) = msg.id {
            conn.send_ack(id, GetStackResponse {
                ok: true,
                stack: StackData {
                    name: stack_name,
                    compose_yaml,
                    compose_env,
                    compose_file_name,
                    is_managed_by_dockge: true,
                },
            }).await;
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

        let stacks_dir = state.config.stacks_dir.clone();
        let stack_dir =
            format!("{}/{}", stacks_dir, stack_name);

        // Save compose files to disk (fast, stays inline)
        {
            let lock = state.stack_locks.get(&stack_name);
            let _guard = lock.lock().await;
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
        }

        // Ack immediately
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

        // Run compose up in background so the worker is free for subsequent messages.
        tokio::spawn(async move {
            let lock = state.stack_locks.get(&stack_name);
            let _guard = lock.lock().await;

            let term_name = format!("compose-{}", stack_name);
            state.terminal_manager.recreate(&term_name, TerminalType::Pty).await;
            state.terminal_manager.write_data(&term_name, b"$ docker compose up -d --remove-orphans\r\n".to_vec());

            match run_pty_to_terminal(
                &state.terminal_manager, &term_name,
                "docker", &["compose", "up", "-d", "--remove-orphans"], Some(&stack_dir),
            ).await {
                Ok(()) => info!(stack = %stack_name, "deploy completed"),
                Err(e) => warn!(stack = %stack_name, "deploy failed: {e}"),
            }
            state.terminal_manager.remove_after(&term_name, Duration::from_secs(30));
        });
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
                OkResponse::simple(),
            )
            .await;
        }

        // Run in background so the worker is free for subsequent messages.
        tokio::spawn(async move {
            let stack_dir = format!("{}/{}", state.config.stacks_dir, stack_name);
            let lock = state.stack_locks.get(&stack_name);
            let _guard = lock.lock().await;

            let term_name = format!("compose-{}", stack_name);
            state.terminal_manager.recreate(&term_name, TerminalType::Pty).await;
            state.terminal_manager.write_data(&term_name, b"$ docker compose down --remove-orphans\r\n".to_vec());

            let _ = run_pty_to_terminal(
                &state.terminal_manager, &term_name,
                "docker", &["compose", "down", "--remove-orphans"], Some(&stack_dir),
            ).await;
            state.terminal_manager.remove_after(&term_name, Duration::from_secs(30));

            if opts.delete_stack_files
                && let Err(e) = std::fs::remove_dir_all(&stack_dir)
            {
                error!(stack = %stack_name, "delete stack files: {e}");
            }
            info!(stack = %stack_name, "stack deleted");
        });
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

        if let Some(id) = msg.id {
            conn.send_ack(id, OkResponse::simple()).await;
        }

        tokio::spawn(async move {
            let stack_dir = format!("{}/{}", state.config.stacks_dir, stack_name);
            let lock = state.stack_locks.get(&stack_name);
            let _guard = lock.lock().await;

            let term_name = format!("compose-{}", stack_name);
            state.terminal_manager.recreate(&term_name, TerminalType::Pty).await;
            state.terminal_manager.write_data(&term_name, b"$ docker compose down -v --remove-orphans\r\n".to_vec());

            let _ = run_pty_to_terminal(
                &state.terminal_manager, &term_name,
                "docker", &["compose", "down", "-v", "--remove-orphans"], Some(&stack_dir),
            ).await;
            state.terminal_manager.remove_after(&term_name, Duration::from_secs(30));

            if let Err(e) = std::fs::remove_dir_all(&stack_dir) {
                error!(stack = %stack_name, "force delete stack: {e}");
            }
            info!(stack = %stack_name, "stack force deleted");
        });
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

        if let Some(id) = msg.id {
            conn.send_ack(id, OkResponse::simple()).await;
        }

        tokio::spawn(async move {
            let stack_dir = format!("{}/{}", state.config.stacks_dir, stack_name);
            let lock = state.stack_locks.get(&stack_name);
            let _guard = lock.lock().await;

            let term_name = format!("compose-{}", stack_name);

            // Pull phase
            state.terminal_manager.recreate(&term_name, TerminalType::Pty).await;
            state.terminal_manager.write_data(&term_name, b"$ docker compose pull\r\n".to_vec());
            let _ = run_pty_to_terminal(
                &state.terminal_manager, &term_name,
                "docker", &["compose", "pull"], Some(&stack_dir),
            ).await;

            // Up phase
            state.terminal_manager.recreate(&term_name, TerminalType::Pty).await;
            state.terminal_manager.write_data(&term_name, b"$ docker compose up -d --remove-orphans\r\n".to_vec());
            match run_pty_to_terminal(
                &state.terminal_manager, &term_name,
                "docker", &["compose", "up", "-d", "--remove-orphans"], Some(&stack_dir),
            ).await {
                Ok(()) => info!(stack = %stack_name, "stack updated"),
                Err(e) => warn!(stack = %stack_name, "stack update failed: {e}"),
            }
            state.terminal_manager.remove_after(&term_name, Duration::from_secs(30));
        });
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

