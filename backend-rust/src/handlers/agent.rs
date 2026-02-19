use crate::handlers::auth::check_login;
use crate::state::AppState;
use serde_json::{json, Value};
use socketioxide::extract::{Data, SocketRef, AckSender};
use std::sync::Arc;
use tracing::debug;

/// Register agent management socket event handlers
pub fn register(socket: &SocketRef, _state: Arc<AppState>) {
    // addAgent (stub)
    {
        socket.on("addAgent", move |_socket: SocketRef, ack: AckSender| {
            ack.send(&json!({ "ok": false, "msg": "Agent management is not yet supported in the Rust backend" })).ok();
        });
    }

    // removeAgent (stub)
    {
        socket.on("removeAgent", move |_socket: SocketRef, ack: AckSender| {
            ack.send(&json!({ "ok": false, "msg": "Agent management is not yet supported in the Rust backend" })).ok();
        });
    }

    // updateAgent (stub)
    {
        socket.on("updateAgent", move |_socket: SocketRef, ack: AckSender| {
            ack.send(&json!({ "ok": false, "msg": "Agent management is not yet supported in the Rust backend" })).ok();
        });
    }
}

/// Register the agent proxy handler.
/// The frontend sends `socket.emit("agent", endpoint, eventName, ...args, callback)`.
/// We need to route this to the correct handler.
pub fn register_agent_proxy(socket: &SocketRef, state: Arc<AppState>) {
    let state = state.clone();
    let _socket_clone = socket.clone();

    socket.on("agent", move |socket: SocketRef, Data(data): Data<Value>, ack: AckSender| {
        let state = state.clone();
        let socket = socket.clone();

        tokio::spawn(async move {
            // data is [endpoint, eventName, ...args]
            let arr = match data.as_array() {
                Some(a) if a.len() >= 2 => a,
                _ => {
                    ack.send(&json!({ "ok": false, "msg": "Invalid agent call" })).ok();
                    return;
                }
            };

            let endpoint = arr[0].as_str().unwrap_or("");
            let event_name = arr[1].as_str().unwrap_or("");
            let remaining_args: Value = Value::Array(arr[2..].to_vec());

            debug!("Agent proxy: endpoint={}, event={}", endpoint, event_name);

            // For now, only handle local (empty endpoint)
            // Re-emit the event directly on this socket so our handlers catch it
            // This is the simplest approach — the handlers are already registered
            // on the socket directly.
            //
            // However, socketioxide doesn't let us re-emit to trigger handlers,
            // so we need to route manually here.

            // Route to the appropriate handler
            route_agent_event(&state, &socket, event_name, &remaining_args, ack).await;
        });
    });
}

/// Route an agent event to the correct handler
async fn route_agent_event(
    state: &Arc<AppState>,
    socket: &SocketRef,
    event: &str,
    args: &Value,
    ack: AckSender,
) {
    match event {
        // Stack operations
        "requestStackList" => {
            if check_login(socket).is_none() {
                ack.send(&json!({ "ok": false, "msg": "Not logged in" })).ok();
                return;
            }
            crate::handlers::stack::refresh_stack_cache(state).await;
            crate::handlers::stack::broadcast_stack_list(state).await;
            ack.send(&json!({ "ok": true, "msg": "Updated", "msgi18n": true })).ok();
        }

        "getStack" => {
            let result = route_with_data(state, socket, args, |s, sk, d| {
                Box::pin(async move { handle_get_stack_routed(&s, &sk, &d).await })
            }).await;
            ack.send(&result).ok();
        }

        "deployStack" | "saveStack" | "startStack" | "stopStack" | "restartStack"
        | "downStack" | "updateStack" | "deleteStack" | "forceDeleteStack"
        | "serviceStatusList" | "startService" | "stopService" | "restartService"
        | "updateService" | "checkImageUpdates" | "getDockerNetworkList"
        | "dockerStats" | "containerInspect"
        | "terminalJoin" | "terminalInput" | "terminalResize"
        | "leaveCombinedTerminal" | "interactiveTerminal" | "joinContainerLog"
        | "mainTerminal" | "checkMainTerminal" => {
            // These events are directly registered on the socket,
            // but since we're routing through the agent proxy,
            // we need to emit them back.
            // The simplest approach is to have all handlers accept
            // both direct and routed calls.
            //
            // For now, emit the event back on the socket itself
            // to trigger the registered handler.
            socket.emit(event, args).ok();
            // Note: this won't trigger on() handlers on the same socket
            // in socketioxide. We need a different approach.
            //
            // The actual solution is to NOT register handlers directly
            // on the socket, and instead only route through the agent proxy.
            // This is what we'll do — see the main.rs on_connect handler.
            ack.send(&json!({ "ok": false, "msg": "Event routing not yet implemented for this event" })).ok();
        }

        _ => {
            debug!("Unknown agent event: {}", event);
            ack.send(&json!({ "ok": false, "msg": format!("Unknown event: {}", event) })).ok();
        }
    }
}

async fn handle_get_stack_routed(state: &Arc<AppState>, socket: &SocketRef, data: &Value) -> Value {
    if check_login(socket).is_none() {
        return json!({ "ok": false, "msg": "Not logged in" });
    }

    let stack_name = if data.is_array() {
        data.get(0).and_then(|v| v.as_str()).unwrap_or("")
    } else {
        data.as_str().unwrap_or("")
    };

    match crate::models::stack::Stack::get_stack(&state.stacks_dir, stack_name).await {
        Ok(s) => {
            let recreate = state.recreate_cache.read().await.get(stack_name).copied().unwrap_or(false);
            let has_updates = state.update_cache.read().await
                .get(stack_name).map(|u| u.has_updates).unwrap_or(false);
            let primary_hostname = crate::models::settings::get(&state.db, "primaryHostname")
                .await.ok().flatten()
                .and_then(|v| v.as_str().map(|s| s.to_string()))
                .unwrap_or_else(|| "localhost".to_string());
            let stack_json = s.to_json("", recreate, has_updates, &primary_hostname).await;
            json!({ "ok": true, "stack": stack_json })
        }
        Err(e) => json!({ "ok": false, "msg": format!("{}", e) }),
    }
}

/// Helper to route with data extraction
async fn route_with_data<F, Fut>(
    state: &Arc<AppState>,
    socket: &SocketRef,
    data: &Value,
    handler: F,
) -> Value
where
    F: FnOnce(Arc<AppState>, SocketRef, Value) -> Fut,
    Fut: std::future::Future<Output = Value>,
{
    handler(state.clone(), socket.clone(), data.clone()).await
}
