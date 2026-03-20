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

    // startStack — uses "up -d --remove-orphans" (not "start") for managed stacks
    ws.handle_with_state("startStack", state.clone(), |state, conn, msg| async move {
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
            let lock = state.stack_locks.get(&stack_name);
            let _guard = lock.lock().await;

            let stack_dir = format!("{}/{}", state.config.stacks_dir, stack_name);
            let term_name = format!("compose-{}", stack_name);
            state.terminal_manager.recreate(&term_name, TerminalType::Pty).await;
            state.terminal_manager.write_data(
                &term_name,
                b"$ docker compose up -d --remove-orphans\r\n".to_vec(),
            );

            match run_pty_to_terminal(
                &state.terminal_manager, &term_name,
                "docker", &["compose", "up", "-d", "--remove-orphans"], Some(&stack_dir),
            ).await {
                Ok(()) => info!(stack = %stack_name, "startStack completed"),
                Err(e) => warn!(stack = %stack_name, "startStack failed: {e}"),
            }
            state.terminal_manager.remove_after(&term_name, Duration::from_secs(30));
        });
    });

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

        // Trigger immediate stacks broadcast so the frontend store is updated
        // before the ack triggers navigation to the new stack page.
        let _ = state.dispatch_tx.try_send(
            crate::broadcast::DispatchMsg::FullSync {
                resource_type: "stacks".to_string(),
            },
        );

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

    // deployStack — validate then deploy; ack AFTER completion so the frontend
    // stays on /stacks/new showing progress output until done.
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

        // Validate then deploy in background; ack AFTER completion so the
        // frontend stays on the current page showing progress output.
        tokio::spawn(async move {
            let lock = state.stack_locks.get(&stack_name);
            let _guard = lock.lock().await;

            let term_name = format!("compose-{}", stack_name);
            state.terminal_manager.recreate(&term_name, TerminalType::Pty).await;

            // Step 1: Validate via dry-run
            state.terminal_manager.write_data(
                &term_name,
                b"$ docker compose config --dry-run\r\n".to_vec(),
            );
            match run_pty_to_terminal(
                &state.terminal_manager, &term_name,
                "docker", &["compose", "config", "--dry-run"], Some(&stack_dir),
            ).await {
                Ok(()) => {}
                Err(e) => {
                    warn!(stack = %stack_name, "deploy validation failed: {e}");
                    // Ack with success — the stack was saved to disk; fsnotify
                    // picks it up. Validation failure just means no "up".
                    if let Some(id) = msg.id {
                        conn.send_ack(id, OkResponse {
                            ok: true,
                            msg: Some("Deployed".into()),
                            token: None,
                        }).await;
                    }
                    state.terminal_manager.remove_after(&term_name, Duration::from_secs(30));
                    return;
                }
            }

            // Step 2: Deploy
            state.terminal_manager.recreate(&term_name, TerminalType::Pty).await;
            state.terminal_manager.write_data(
                &term_name,
                b"$ docker compose up -d --remove-orphans\r\n".to_vec(),
            );
            match run_pty_to_terminal(
                &state.terminal_manager, &term_name,
                "docker", &["compose", "up", "-d", "--remove-orphans"], Some(&stack_dir),
            ).await {
                Ok(()) => info!(stack = %stack_name, "deploy completed"),
                Err(e) => warn!(stack = %stack_name, "deploy failed: {e}"),
            }

            // Ack after completion
            if let Some(id) = msg.id {
                conn.send_ack(id, OkResponse {
                    ok: true,
                    msg: Some("Deployed".into()),
                    token: None,
                }).await;
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

#[cfg(test)]
mod tests {
    use super::*;

    // ── validate_stack_name ─────────────────────────────────────────────

    #[test]
    fn validate_empty_name() {
        assert_eq!(validate_stack_name(""), Err("Stack name required"));
    }

    #[test]
    fn validate_valid_names() {
        assert!(validate_stack_name("my-stack").is_ok());
        assert!(validate_stack_name("web-app-2").is_ok());
        assert!(validate_stack_name("a").is_ok());
        assert!(validate_stack_name("test123").is_ok());
    }

    #[test]
    fn validate_path_traversal_dotdot() {
        assert_eq!(validate_stack_name(".."), Err("Invalid stack name (path traversal)"));
    }

    #[test]
    fn validate_path_traversal_embedded() {
        assert_eq!(validate_stack_name("foo/../etc"), Err("Invalid stack name (path traversal)"));
    }

    #[test]
    fn validate_path_traversal_slash() {
        assert_eq!(validate_stack_name("foo/bar"), Err("Invalid stack name (path traversal)"));
    }

    #[test]
    fn validate_path_traversal_backslash() {
        assert_eq!(validate_stack_name("foo\\bar"), Err("Invalid stack name (path traversal)"));
    }

    #[test]
    fn validate_null_byte() {
        assert_eq!(validate_stack_name("foo\0bar"), Err("Invalid stack name (null byte)"));
    }

    #[test]
    fn validate_shell_chars() {
        for ch in &[';', '|', '&', '`', '$', '>', '<', '(', ')'] {
            let name = format!("foo{}bar", ch);
            assert_eq!(
                validate_stack_name(&name),
                Err("Invalid stack name (shell characters)"),
                "should reject '{ch}'"
            );
        }
    }

    #[test]
    fn validate_dot_prefix() {
        assert_eq!(validate_stack_name(".hidden"), Err("Invalid stack name (dot prefix)"));
    }

    #[test]
    fn validate_leading_hyphen() {
        assert_eq!(validate_stack_name("-flag"), Err("Invalid stack name (leading hyphen)"));
    }

    #[test]
    fn validate_spaces() {
        assert_eq!(validate_stack_name("my stack"), Err("Invalid stack name (spaces)"));
    }

    #[test]
    fn validate_uppercase() {
        assert_eq!(validate_stack_name("MyStack"), Err("Invalid stack name (uppercase)"));
    }

    // ── NamedMutex ──────────────────────────────────────────────────────

    #[test]
    fn named_mutex_same_name_returns_same_arc() {
        let nm = NamedMutex::new();
        let a = nm.get("test");
        let b = nm.get("test");
        assert!(Arc::ptr_eq(&a, &b));
    }

    #[test]
    fn named_mutex_different_names_return_different_arcs() {
        let nm = NamedMutex::new();
        let a = nm.get("alpha");
        let b = nm.get("beta");
        assert!(!Arc::ptr_eq(&a, &b));
    }

    #[test]
    fn named_mutex_fresh_name_gets_new_mutex() {
        let nm = NamedMutex::new();
        let _a = nm.get("first");
        let b = nm.get("second");
        let c = nm.get("second");
        assert!(Arc::ptr_eq(&b, &c));
    }

    // ── save_stack_to_disk (filesystem) ─────────────────────────────────

    #[test]
    fn save_creates_dir_and_compose_file() {
        let tmp = tempfile::tempdir().unwrap();
        let stack_dir = tmp.path().join("my-stack");
        save_stack_to_disk(stack_dir.to_str().unwrap(), "version: '3'\n", "").unwrap();
        let content = std::fs::read_to_string(stack_dir.join("compose.yaml")).unwrap();
        assert_eq!(content, "version: '3'\n");
    }

    #[test]
    fn save_writes_env_when_nonempty() {
        let tmp = tempfile::tempdir().unwrap();
        let stack_dir = tmp.path().join("my-stack");
        save_stack_to_disk(stack_dir.to_str().unwrap(), "v: 3\n", "FOO=bar").unwrap();
        let env = std::fs::read_to_string(stack_dir.join(".env")).unwrap();
        assert_eq!(env, "FOO=bar");
    }

    #[test]
    fn save_removes_env_when_empty() {
        let tmp = tempfile::tempdir().unwrap();
        let stack_dir = tmp.path().join("my-stack");
        save_stack_to_disk(stack_dir.to_str().unwrap(), "v: 3\n", "FOO=bar").unwrap();
        assert!(stack_dir.join(".env").exists());
        save_stack_to_disk(stack_dir.to_str().unwrap(), "v: 3\n", "").unwrap();
        assert!(!stack_dir.join(".env").exists());
    }

    // ── find_compose_file (filesystem) ──────────────────────────────────

    #[test]
    fn find_compose_file_yaml() {
        let tmp = tempfile::tempdir().unwrap();
        let stack_dir = tmp.path().join("test-stack");
        std::fs::create_dir_all(&stack_dir).unwrap();
        std::fs::write(stack_dir.join("compose.yaml"), "").unwrap();
        assert_eq!(
            find_compose_file(tmp.path().to_str().unwrap(), "test-stack"),
            Some("compose.yaml".to_string())
        );
    }

    #[test]
    fn find_compose_file_docker_compose_yml() {
        let tmp = tempfile::tempdir().unwrap();
        let stack_dir = tmp.path().join("test-stack");
        std::fs::create_dir_all(&stack_dir).unwrap();
        std::fs::write(stack_dir.join("docker-compose.yml"), "").unwrap();
        assert_eq!(
            find_compose_file(tmp.path().to_str().unwrap(), "test-stack"),
            Some("docker-compose.yml".to_string())
        );
    }

    #[test]
    fn find_compose_file_missing() {
        let tmp = tempfile::tempdir().unwrap();
        let stack_dir = tmp.path().join("empty-stack");
        std::fs::create_dir_all(&stack_dir).unwrap();
        assert_eq!(
            find_compose_file(tmp.path().to_str().unwrap(), "empty-stack"),
            None
        );
    }
}

