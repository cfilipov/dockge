use crate::docker;
use crate::error::{error_response, ok_response_i18n};
use crate::handlers::auth::check_login;
use crate::models::settings as settings_model;
use crate::models::stack::Stack;
use crate::state::AppState;
use crate::terminal::{
    get_combined_terminal_name, get_compose_terminal_name, TerminalManager,
};
use crate::update_checker;
use serde_json::{json, Value};
use socketioxide::extract::{Data, SocketRef, AckSender};
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::broadcast;
use tracing::{debug, warn};

/// Global terminal manager
static TERMINAL_MANAGER: std::sync::LazyLock<TerminalManager> =
    std::sync::LazyLock::new(TerminalManager::new);

/// Register stack-related socket event handlers via the agent proxy
pub fn register_agent_handlers(socket: &SocketRef, state: Arc<AppState>) {
    // requestStackList
    {
        let state = state.clone();
        socket.on("requestStackList", move |socket: SocketRef, ack: AckSender| {
            let state = state.clone();
            tokio::spawn(async move {
                if check_login(&socket).is_none() {
                    ack.send(&error_response("Not logged in")).ok();
                    return;
                }
                refresh_stack_cache(&state).await;
                broadcast_stack_list(&state).await;
                ack.send(&ok_response_i18n("Updated")).ok();
            });
        });
    }

    // getStack
    {
        let state = state.clone();
        socket.on("getStack", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let state = state.clone();
            tokio::spawn(async move {
                let result = handle_get_stack(&state, &socket, &data).await;
                ack.send(&result).ok();
            });
        });
    }

    // deployStack
    {
        let state = state.clone();
        socket.on("deployStack", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let state = state.clone();
            let socket = socket.clone();
            tokio::spawn(async move {
                let result = handle_deploy_stack(&state, &socket, &data).await;
                ack.send(&result).ok();
            });
        });
    }

    // saveStack
    {
        let state = state.clone();
        socket.on("saveStack", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let state = state.clone();
            tokio::spawn(async move {
                let result = handle_save_stack(&state, &socket, &data).await;
                ack.send(&result).ok();
            });
        });
    }

    // startStack
    {
        let state = state.clone();
        socket.on("startStack", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let state = state.clone();
            let socket = socket.clone();
            tokio::spawn(async move {
                let result = handle_compose_action(&state, &socket, &data, "up", &["-d", "--remove-orphans"]).await;
                ack.send(&result).ok();
            });
        });
    }

    // stopStack
    {
        let state = state.clone();
        socket.on("stopStack", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let state = state.clone();
            let socket = socket.clone();
            tokio::spawn(async move {
                let result = handle_compose_action(&state, &socket, &data, "stop", &[]).await;
                ack.send(&result).ok();
            });
        });
    }

    // restartStack
    {
        let state = state.clone();
        socket.on("restartStack", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let state = state.clone();
            let socket = socket.clone();
            tokio::spawn(async move {
                let result = handle_compose_action(&state, &socket, &data, "restart", &[]).await;
                ack.send(&result).ok();
            });
        });
    }

    // downStack
    {
        let state = state.clone();
        socket.on("downStack", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let state = state.clone();
            let socket = socket.clone();
            tokio::spawn(async move {
                let result = handle_compose_action(&state, &socket, &data, "down", &[]).await;
                ack.send(&result).ok();
            });
        });
    }

    // updateStack
    {
        let state = state.clone();
        socket.on("updateStack", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let state = state.clone();
            let socket = socket.clone();
            tokio::spawn(async move {
                let result = handle_update_stack(&state, &socket, &data).await;
                ack.send(&result).ok();
            });
        });
    }

    // deleteStack
    {
        let state = state.clone();
        socket.on("deleteStack", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let state = state.clone();
            let socket = socket.clone();
            tokio::spawn(async move {
                let result = handle_delete_stack(&state, &socket, &data, false).await;
                ack.send(&result).ok();
            });
        });
    }

    // forceDeleteStack
    {
        let state = state.clone();
        socket.on("forceDeleteStack", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let state = state.clone();
            let socket = socket.clone();
            tokio::spawn(async move {
                let result = handle_delete_stack(&state, &socket, &data, true).await;
                ack.send(&result).ok();
            });
        });
    }

    // serviceStatusList
    {
        let state = state.clone();
        socket.on("serviceStatusList", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let state = state.clone();
            tokio::spawn(async move {
                let result = handle_service_status_list(&state, &socket, &data).await;
                ack.send(&result).ok();
            });
        });
    }

    // startService / stopService / restartService
    for (event, action) in [
        ("startService", "up"),
        ("stopService", "stop"),
        ("restartService", "restart"),
    ] {
        let state = state.clone();
        let action = action.to_string();
        socket.on(event, move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let state = state.clone();
            let socket = socket.clone();
            let action = action.clone();
            tokio::spawn(async move {
                let result = handle_service_action(&state, &socket, &data, &action).await;
                ack.send(&result).ok();
            });
        });
    }

    // updateService
    {
        let state = state.clone();
        socket.on("updateService", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let state = state.clone();
            let socket = socket.clone();
            tokio::spawn(async move {
                let result = handle_update_service(&state, &socket, &data).await;
                ack.send(&result).ok();
            });
        });
    }

    // checkImageUpdates
    {
        let state = state.clone();
        socket.on("checkImageUpdates", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let state = state.clone();
            tokio::spawn(async move {
                let result = handle_check_image_updates(&state, &socket, &data).await;
                ack.send(&result).ok();
            });
        });
    }

    // getDockerNetworkList
    {
        let state = state.clone();
        socket.on("getDockerNetworkList", move |socket: SocketRef, ack: AckSender| {
            let _state = state.clone();
            tokio::spawn(async move {
                if check_login(&socket).is_none() {
                    ack.send(&error_response("Not logged in")).ok();
                    return;
                }
                match docker::get_network_list().await {
                    Ok(list) => { ack.send(&json!({ "ok": true, "dockerNetworkList": list })).ok(); }
                    Err(e) => { ack.send(&error_response(&format!("{}", e))).ok(); }
                }
            });
        });
    }

    // dockerStats
    {
        let state = state.clone();
        socket.on("dockerStats", move |socket: SocketRef, ack: AckSender| {
            let _state = state.clone();
            tokio::spawn(async move {
                if check_login(&socket).is_none() {
                    ack.send(&error_response("Not logged in")).ok();
                    return;
                }
                match docker::get_docker_stats().await {
                    Ok(stats) => { ack.send(&json!({ "ok": true, "dockerStats": stats })).ok(); }
                    Err(e) => { ack.send(&error_response(&format!("{}", e))).ok(); }
                }
            });
        });
    }

    // containerInspect
    {
        let state = state.clone();
        socket.on("containerInspect", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
            let _state = state.clone();
            tokio::spawn(async move {
                if check_login(&socket).is_none() {
                    ack.send(&error_response("Not logged in")).ok();
                    return;
                }
                let container_name = if data.is_array() {
                    data.get(0).and_then(|v| v.as_str()).unwrap_or("")
                } else {
                    data.as_str().unwrap_or("")
                };
                match docker::container_inspect(container_name).await {
                    Ok(inspect_data) => {
                        ack.send(&json!({ "ok": true, "inspectData": inspect_data })).ok();
                    }
                    Err(e) => { ack.send(&error_response(&format!("{}", e))).ok(); }
                }
            });
        });
    }
}

/// Refresh the in-memory stack cache
pub async fn refresh_stack_cache(state: &Arc<AppState>) {
    match Stack::get_stack_list(&state.stacks_dir).await {
        Ok(stacks) => {
            let update_cache = state.update_cache.read().await;
            let recreate_cache = state.recreate_cache.read().await;

            let mut cache = HashMap::new();
            for (name, stack_obj) in &stacks {
                let recreate = recreate_cache.get(name).copied().unwrap_or(false);
                let has_updates = update_cache
                    .get(name)
                    .map(|u| u.has_updates)
                    .unwrap_or(false);

                cache.insert(
                    name.clone(),
                    stack_obj.to_simple_json("", recreate, has_updates),
                );
            }

            let mut stack_cache = state.stack_cache.write().await;
            *stack_cache = cache;
        }
        Err(e) => {
            warn!("Failed to refresh stack cache: {}", e);
        }
    }
}

/// Broadcast stack list to all authenticated sockets
pub async fn broadcast_stack_list(state: &Arc<AppState>) {
    refresh_stack_cache(state).await;

    let stack_cache = state.stack_cache.read().await;
    let stack_list: serde_json::Map<String, Value> = stack_cache
        .iter()
        .map(|(k, v)| (k.clone(), serde_json::to_value(v).unwrap_or_default()))
        .collect();

    let data = json!({
        "ok": true,
        "stackList": stack_list,
        "endpoint": "",
    });

    // Broadcast to all sockets in the "/" namespace
    state.io.emit("agent", &("stackList", &data)).ok();
}

pub async fn handle_get_stack(state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
    if check_login(socket).is_none() {
        return error_response("Not logged in");
    }

    let stack_name = if data.is_array() {
        data.get(0).and_then(|v| v.as_str()).unwrap_or("")
    } else {
        data.as_str().unwrap_or("")
    };

    if stack_name.is_empty() {
        return error_response("Stack name is required");
    }

    match Stack::get_stack(&state.stacks_dir, stack_name).await {
        Ok(s) => {
            let recreate = state.recreate_cache.read().await.get(stack_name).copied().unwrap_or(false);
            let has_updates = state.update_cache.read().await
                .get(stack_name)
                .map(|u| u.has_updates)
                .unwrap_or(false);

            let primary_hostname = settings_model::get(&state.db, "primaryHostname")
                .await
                .ok()
                .flatten()
                .and_then(|v| v.as_str().map(|s| s.to_string()))
                .unwrap_or_else(|| "localhost".to_string());

            let stack_json = s.to_json("", recreate, has_updates, &primary_hostname).await;

            // Join combined terminal if managed
            if s.is_managed_by_dockge() {
                join_combined_terminal(state, socket, "", stack_name).await;
            }

            json!({ "ok": true, "stack": stack_json })
        }
        Err(e) => error_response(&format!("{}", e)),
    }
}

pub async fn handle_deploy_stack(state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
    if check_login(socket).is_none() {
        return error_response("Not logged in");
    }

    let arr = match data.as_array() {
        Some(a) => a,
        None => return error_response("Invalid data"),
    };

    let name = arr.first().and_then(|v| v.as_str()).unwrap_or("");
    let compose_yaml = arr.get(1).and_then(|v| v.as_str()).unwrap_or("");
    let compose_env = arr.get(2).and_then(|v| v.as_str()).unwrap_or("");
    let compose_override_yaml = arr.get(3).and_then(|v| v.as_str()).unwrap_or("");
    let is_add = arr.get(4).and_then(|v| v.as_bool()).unwrap_or(false);

    let mut stack_obj = Stack::new(name.to_string(), state.stacks_dir.clone());
    stack_obj.compose_yaml = compose_yaml.to_string();
    stack_obj.compose_env = compose_env.to_string();
    stack_obj.compose_override_yaml = compose_override_yaml.to_string();

    if let Err(e) = stack_obj.save(is_add).await {
        return error_response(&format!("{}", e));
    }

    // Run docker compose up -d
    let terminal_name = get_compose_terminal_name("", name);
    let args = stack_obj.get_compose_options("up", &["-d", "--remove-orphans"]);
    let (tx, _) = broadcast::channel(256);
    let socket_clone = socket.clone();
    let _terminal_name_clone = terminal_name.clone();
    let mut rx = tx.subscribe();

    // Stream output to socket
    let output_task = tokio::spawn(async move {
        while let Ok((term_name, data)) = rx.recv().await {
            socket_clone.emit("agent", &("terminalWrite", &term_name, &data)).ok();
        }
    });

    let exit_code = TERMINAL_MANAGER
        .exec(&terminal_name, "docker", &args, &stack_obj.path(), tx)
        .await;

    drop(output_task);

    match exit_code {
        Ok(0) => {
            refresh_stack_cache(state).await;
            broadcast_stack_list(state).await;
            join_combined_terminal(state, socket, "", name).await;
            ok_response_i18n("Deployed")
        }
        Ok(code) => error_response(&format!("Failed to deploy (exit code {})", code)),
        Err(e) => error_response(&e),
    }
}

pub async fn handle_save_stack(state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
    if check_login(socket).is_none() {
        return error_response("Not logged in");
    }

    let arr = match data.as_array() {
        Some(a) => a,
        None => return error_response("Invalid data"),
    };

    let name = arr.first().and_then(|v| v.as_str()).unwrap_or("");
    let compose_yaml = arr.get(1).and_then(|v| v.as_str()).unwrap_or("");
    let compose_env = arr.get(2).and_then(|v| v.as_str()).unwrap_or("");
    let compose_override_yaml = arr.get(3).and_then(|v| v.as_str()).unwrap_or("");
    let is_add = arr.get(4).and_then(|v| v.as_bool()).unwrap_or(false);

    let mut stack_obj = Stack::new(name.to_string(), state.stacks_dir.clone());
    stack_obj.compose_yaml = compose_yaml.to_string();
    stack_obj.compose_env = compose_env.to_string();
    stack_obj.compose_override_yaml = compose_override_yaml.to_string();

    match stack_obj.save(is_add).await {
        Ok(()) => {
            refresh_stack_cache(state).await;
            broadcast_stack_list(state).await;
            ok_response_i18n("Saved")
        }
        Err(e) => error_response(&format!("{}", e)),
    }
}

/// Handle generic compose actions (start, stop, restart, down)
pub async fn handle_compose_action(
    state: &Arc<AppState>,
    socket: &SocketRef,
    data: &Value,
    command: &str,
    extra_args: &[&str],
) -> Value {
    if check_login(socket).is_none() {
        return error_response("Not logged in");
    }

    let stack_name = if data.is_array() {
        data.get(0).and_then(|v| v.as_str()).unwrap_or("")
    } else {
        data.as_str().unwrap_or("")
    };

    if stack_name.is_empty() {
        return error_response("Stack name is required");
    }

    let stack_obj = match Stack::get_stack(&state.stacks_dir, stack_name).await {
        Ok(s) => s,
        Err(e) => return error_response(&format!("{}", e)),
    };

    let terminal_name = get_compose_terminal_name("", stack_name);
    let args = stack_obj.get_compose_options(command, extra_args);
    let (tx, _) = broadcast::channel(256);
    let socket_clone = socket.clone();
    let mut rx = tx.subscribe();

    let output_task = tokio::spawn(async move {
        while let Ok((term_name, data)) = rx.recv().await {
            socket_clone.emit("agent", &("terminalWrite", &term_name, &data)).ok();
        }
    });

    let exit_code = TERMINAL_MANAGER
        .exec(&terminal_name, "docker", &args, &stack_obj.path(), tx)
        .await;

    drop(output_task);

    // Refresh after action
    refresh_stack_cache(state).await;
    broadcast_stack_list(state).await;

    let msg = match command {
        "up" => "Started",
        "stop" => "Stopped",
        "restart" => "Restarted",
        "down" => "Downed",
        _ => "Done",
    };

    match exit_code {
        Ok(0) => ok_response_i18n(msg),
        Ok(code) => error_response(&format!("Failed (exit code {})", code)),
        Err(e) => error_response(&e),
    }
}

pub async fn handle_update_stack(state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
    if check_login(socket).is_none() {
        return error_response("Not logged in");
    }

    let stack_name = if data.is_array() {
        data.get(0).and_then(|v| v.as_str()).unwrap_or("")
    } else {
        data.as_str().unwrap_or("")
    };

    let stack_obj = match Stack::get_stack(&state.stacks_dir, stack_name).await {
        Ok(s) => s,
        Err(e) => return error_response(&format!("{}", e)),
    };

    let terminal_name = get_compose_terminal_name("", stack_name);
    let (tx, _) = broadcast::channel(256);
    let socket_clone = socket.clone();
    let mut rx = tx.subscribe();

    let output_task = tokio::spawn(async move {
        while let Ok((term_name, data)) = rx.recv().await {
            socket_clone.emit("agent", &("terminalWrite", &term_name, &data)).ok();
        }
    });

    // Step 1: pull
    let pull_args = stack_obj.get_compose_options("pull", &[]);
    let exit_code = TERMINAL_MANAGER
        .exec(&terminal_name, "docker", &pull_args, &stack_obj.path(), tx.clone())
        .await;

    if exit_code != Ok(0) {
        drop(output_task);
        return error_response("Failed to pull images");
    }

    // Step 2: check if running, then up
    if stack_obj.is_started() {
        let up_args = stack_obj.get_compose_options("up", &["-d", "--remove-orphans"]);
        let terminal_name2 = format!("{}-up", terminal_name);
        let exit_code = TERMINAL_MANAGER
            .exec(&terminal_name2, "docker", &up_args, &stack_obj.path(), tx.clone())
            .await;

        if exit_code != Ok(0) {
            drop(output_task);
            return error_response("Failed to recreate containers");
        }

        // Step 3: prune images
        let prune_args = vec![
            "image".to_string(),
            "prune".to_string(),
            "--all".to_string(),
            "--force".to_string(),
        ];
        let terminal_name3 = format!("{}-prune", terminal_name);
        let _ = TERMINAL_MANAGER
            .exec(&terminal_name3, "docker", &prune_args, &stack_obj.path(), tx)
            .await;
    }

    drop(output_task);

    refresh_stack_cache(state).await;
    broadcast_stack_list(state).await;

    json!({ "ok": true, "msg": format!("Updated {}", stack_name), "msgi18n": true })
}

pub async fn handle_delete_stack(state: &Arc<AppState>, socket: &SocketRef, data: &Value, force: bool) -> Value {
    if check_login(socket).is_none() {
        return error_response("Not logged in");
    }

    let (stack_name, delete_files) = if data.is_array() {
        let arr = data.as_array().unwrap();
        let name = arr.first().and_then(|v| v.as_str()).unwrap_or("");
        if force {
            (name, true)
        } else {
            let opts = arr.get(1);
            let delete = opts
                .and_then(|v| v.get("deleteStackFiles"))
                .and_then(|v| v.as_bool())
                .unwrap_or(false);
            (name, delete)
        }
    } else {
        (data.as_str().unwrap_or(""), force)
    };

    let stack_obj = match Stack::get_stack(&state.stacks_dir, stack_name).await {
        Ok(s) => s,
        Err(e) => return error_response(&format!("{}", e)),
    };

    let terminal_name = get_compose_terminal_name("", stack_name);
    let down_extra = if force { vec!["-v", "--remove-orphans"] } else { vec!["--remove-orphans"] };
    let args = stack_obj.get_compose_options("down", &down_extra);
    let (tx, _) = broadcast::channel(256);
    let socket_clone = socket.clone();
    let mut rx = tx.subscribe();

    let output_task = tokio::spawn(async move {
        while let Ok((term_name, data)) = rx.recv().await {
            socket_clone.emit("agent", &("terminalWrite", &term_name, &data)).ok();
        }
    });

    let _ = TERMINAL_MANAGER
        .exec(&terminal_name, "docker", &args, &stack_obj.path(), tx)
        .await;

    drop(output_task);

    // Delete files if requested
    if delete_files || force {
        if let Err(e) = tokio::fs::remove_dir_all(stack_obj.path()).await {
            warn!("Failed to delete stack files: {}", e);
        }
    }

    refresh_stack_cache(state).await;
    broadcast_stack_list(state).await;

    ok_response_i18n("Deleted")
}

pub async fn handle_service_status_list(state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
    if check_login(socket).is_none() {
        return error_response("Not logged in");
    }

    let stack_name = if data.is_array() {
        data.get(0).and_then(|v| v.as_str()).unwrap_or("")
    } else {
        data.as_str().unwrap_or("")
    };

    let stack_obj = match Stack::get_stack(&state.stacks_dir, stack_name).await {
        Ok(s) => s,
        Err(e) => return error_response(&format!("{}", e)),
    };

    match stack_obj.get_service_status_list().await {
        Ok(status_list) => {
            // Compute recreateNecessary
            let mut any_recreate = false;
            let compose_doc: Value = serde_yaml::from_str(&stack_obj.compose_yaml).unwrap_or_default();
            let services = compose_doc.get("services").and_then(|s| s.as_object());

            let mut recreate_status = serde_json::Map::new();

            if let Some(services) = services {
                for (svc_name, entries) in &status_list {
                    if let Some(compose_svc) = services.get(svc_name) {
                        let compose_image = compose_svc.get("image").and_then(|i| i.as_str()).unwrap_or("");
                        if !compose_image.is_empty() {
                            if let Some(first_entry) = entries.first() {
                                let running_image = first_entry.get("image").and_then(|i| i.as_str()).unwrap_or("");
                                if !running_image.is_empty() && running_image != compose_image {
                                    any_recreate = true;
                                    recreate_status.insert(svc_name.clone(), json!(true));
                                }
                            }
                        }
                    }
                }
            }

            // Update recreate cache
            {
                let mut cache = state.recreate_cache.write().await;
                cache.insert(stack_name.to_string(), any_recreate);
            }

            // Get update status
            let update_cache = state.update_cache.read().await;
            let service_update_status = update_cache.get(stack_name).map(|entry| {
                let mut map = serde_json::Map::new();
                for (svc, has_update) in &entry.services {
                    map.insert(svc.clone(), json!({ "hasUpdate": has_update }));
                }
                map
            }).unwrap_or_default();

            json!({
                "ok": true,
                "serviceStatusList": status_list,
                "serviceUpdateStatus": service_update_status,
                "serviceRecreateStatus": recreate_status,
            })
        }
        Err(e) => error_response(&format!("{}", e)),
    }
}

pub async fn handle_service_action(
    state: &Arc<AppState>,
    socket: &SocketRef,
    data: &Value,
    action: &str,
) -> Value {
    if check_login(socket).is_none() {
        return error_response("Not logged in");
    }

    let arr = match data.as_array() {
        Some(a) => a,
        None => return error_response("Invalid data"),
    };

    let stack_name = arr.first().and_then(|v| v.as_str()).unwrap_or("");
    let service_name = arr.get(1).and_then(|v| v.as_str()).unwrap_or("");

    if stack_name.is_empty() || service_name.is_empty() {
        return error_response("Stack name and service name required");
    }

    let stack_obj = match Stack::get_stack(&state.stacks_dir, stack_name).await {
        Ok(s) => s,
        Err(e) => return error_response(&format!("{}", e)),
    };

    let terminal_name = get_compose_terminal_name("", stack_name);
    let args = if action == "up" {
        vec!["compose".to_string(), "up".to_string(), "-d".to_string(), service_name.to_string()]
    } else {
        vec!["compose".to_string(), action.to_string(), service_name.to_string()]
    };

    let (tx, _) = broadcast::channel(256);
    let socket_clone = socket.clone();
    let mut rx = tx.subscribe();

    let output_task = tokio::spawn(async move {
        while let Ok((term_name, data)) = rx.recv().await {
            socket_clone.emit("agent", &("terminalWrite", &term_name, &data)).ok();
        }
    });

    let exit_code = TERMINAL_MANAGER
        .exec(&terminal_name, "docker", &args, &stack_obj.path(), tx)
        .await;

    drop(output_task);

    refresh_stack_cache(state).await;
    broadcast_stack_list(state).await;

    match exit_code {
        Ok(0) => json!({ "ok": true, "msg": format!("Service {} {}ed", service_name, action) }),
        Ok(code) => error_response(&format!("Failed (exit code {})", code)),
        Err(e) => error_response(&e),
    }
}

pub async fn handle_update_service(state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
    if check_login(socket).is_none() {
        return error_response("Not logged in");
    }

    let arr = match data.as_array() {
        Some(a) => a,
        None => return error_response("Invalid data"),
    };

    let stack_name = arr.first().and_then(|v| v.as_str()).unwrap_or("");
    let service_name = arr.get(1).and_then(|v| v.as_str()).unwrap_or("");

    let stack_obj = match Stack::get_stack(&state.stacks_dir, stack_name).await {
        Ok(s) => s,
        Err(e) => return error_response(&format!("{}", e)),
    };

    let terminal_name = get_compose_terminal_name("", stack_name);
    let (tx, _) = broadcast::channel(256);
    let socket_clone = socket.clone();
    let mut rx = tx.subscribe();

    let output_task = tokio::spawn(async move {
        while let Ok((term_name, data)) = rx.recv().await {
            socket_clone.emit("agent", &("terminalWrite", &term_name, &data)).ok();
        }
    });

    // Pull service image
    let pull_args = vec![
        "compose".to_string(), "pull".to_string(), service_name.to_string(),
    ];
    let _ = TERMINAL_MANAGER
        .exec(&terminal_name, "docker", &pull_args, &stack_obj.path(), tx.clone())
        .await;

    // Recreate the service
    let up_args = vec![
        "compose".to_string(), "up".to_string(), "-d".to_string(),
        "--no-deps".to_string(), service_name.to_string(),
    ];
    let terminal_name2 = format!("{}-up", terminal_name);
    let _ = TERMINAL_MANAGER
        .exec(&terminal_name2, "docker", &up_args, &stack_obj.path(), tx)
        .await;

    drop(output_task);

    refresh_stack_cache(state).await;
    broadcast_stack_list(state).await;

    json!({ "ok": true, "msg": format!("Updated {}", service_name) })
}

pub async fn handle_check_image_updates(state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
    if check_login(socket).is_none() {
        return error_response("Not logged in");
    }

    let stack_name = if data.is_array() {
        data.get(0).and_then(|v| v.as_str()).unwrap_or("")
    } else {
        data.as_str().unwrap_or("")
    };

    if stack_name.is_empty() {
        return error_response("Stack name required");
    }

    update_checker::check_stack(state, stack_name).await;
    broadcast_stack_list(state).await;

    json!({ "ok": true, "msg": "Image update check complete" })
}

/// Join the combined terminal for a stack
async fn join_combined_terminal(_state: &Arc<AppState>, _socket: &SocketRef, endpoint: &str, stack_name: &str) {
    // This is a simplified version — in the full implementation,
    // we'd create a persistent terminal running `docker compose logs -f`
    // and relay output to the socket.
    let terminal_name = get_combined_terminal_name(endpoint, stack_name);
    debug!("Socket joined combined terminal: {}", terminal_name);
}
