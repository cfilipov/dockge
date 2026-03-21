// redb uses separate error types (TransactionError, TableError, etc.) with no unified enum;
// individual variants are unavoidably large, so boxing each would add noise for no benefit.
#![allow(clippy::result_large_err)]

mod auth;
mod broadcast;
mod compose;
mod config;
mod db;
mod debug;
mod docker;
mod embed;
mod handlers;
mod terminal;
mod ws;

use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;

use axum::{
    extract::WebSocketUpgrade,
    http::StatusCode,
    response::Json,
    routing::{get, post},
    Router,
};
use clap::Parser;
use config::{Cli, Config};
use std::net::SocketAddr;
use tokio::net::TcpListener;
use tokio::signal;
use tokio::sync::mpsc;
use tower_http::compression::CompressionLayer;
use tracing::{error, info, warn};
use ws::WsServer;

use handlers::AppState;

#[tokio::main(flavor = "multi_thread", worker_threads = 2)]
async fn main() {
    let cli = Cli::parse();
    let config = Config::from(&cli);

    // Initialize tracing
    let filter = match config.log_level.as_str() {
        "debug" => "debug",
        "warn" => "warn",
        "error" => "error",
        _ => "info",
    };
    tracing_subscriber::fmt()
        .with_env_filter(filter)
        .with_target(false)
        .init();

    info!(port = config.port, dev = config.dev, "starting dockge-rust");

    // Ensure directories exist
    if let Err(e) = std::fs::create_dir_all(&config.data_dir) {
        warn!("failed to create data dir: {e}");
    }
    if let Err(e) = std::fs::create_dir_all(&config.stacks_dir) {
        warn!("failed to create stacks dir: {e}");
    }

    // Open database
    let database = Arc::new(
        db::open(std::path::Path::new(&config.data_dir)).expect("failed to open database"),
    );

    let users = db::users::UserStore::new(database.clone());

    // Generate or load JWT secret
    let jwt_secret = match get_or_create_setting(&database, "jwtSecret") {
        Some(s) => s,
        None => {
            let secret = auth::gen_secret(64);
            set_setting(&database, "jwtSecret", &secret);
            secret
        }
    };

    // Determine if setup is needed
    let need_setup = users.count().unwrap_or(0) == 0;

    // Dev mode: auto-seed admin user
    if config.dev && need_setup {
        match users.create("admin", "testpass123") {
            Ok(_) => info!("dev mode: created admin user"),
            Err(e) => error!("dev mode: failed to create admin: {e}"),
        }
    }

    let need_setup_after_seed = users.count().unwrap_or(0) == 0;

    // Connect to Docker (returns DockerClient with automatic timeouts)
    let docker_client = docker::connect().expect("failed to connect to Docker");

    // Create channels
    let broadcaster = broadcast::Broadcaster::new(256);
    let event_bus = broadcast::eventbus::EventBus::new(256);
    let (dispatch_tx, dispatch_rx) = mpsc::channel::<broadcast::DispatchMsg>(256);
    let (ws_control_tx, ws_control_rx) = mpsc::channel::<broadcast::WsControlMsg>(16);

    // Create state
    let state = Arc::new(AppState {
        config: config.clone(),
        db: database.clone(),
        users,
        jwt_secret,
        need_setup: AtomicBool::new(need_setup_after_seed),
        login_limiter: auth::LoginRateLimiter::new(),
        broadcaster: broadcaster.clone(),
        dispatch_tx,
        ws_control_tx,
        docker: docker_client,
        stack_locks: handlers::stack::NamedMutex::new(),
        has_authenticated: AtomicBool::new(false),
        terminal_manager: terminal::spawn(),
        event_bus,
        event_watcher_ready: AtomicBool::new(false),
    });

    // Build WebSocket server with handlers
    let mut ws_builder = WsServer::new(broadcaster);

    // Register connect handler: send "info" event synchronously so it is
    // queued in the write channel before the connection task starts.
    let dev = config.dev;
    let connect_state = state.clone();
    ws_builder.handle_connect(move |conn| {
        #[derive(serde::Serialize)]
        #[serde(rename_all = "camelCase")]
        struct InfoEvent {
            version: &'static str,
            latest_version: &'static str,
            is_container: bool,
            dev: bool,
        }

        conn.send_event_sync("info", InfoEvent {
            version: env!("CARGO_PKG_VERSION"),
            latest_version: env!("CARGO_PKG_VERSION"),
            is_container: true,
            dev,
        });

        // No-auth mode: auto-authenticate and send initial data
        if connect_state.config.no_auth {
            conn.set_user(1);
            connect_state.has_authenticated.store(true, Ordering::Relaxed);
            conn.send_event_sync("autoLogin", serde_json::Value::Null);
            handlers::auth::after_login(&connect_state, &conn);
        }

        // If no users exist, tell the client to show the setup page
        if connect_state.need_setup.load(std::sync::atomic::Ordering::Relaxed) {
            conn.send_event_sync("setup", serde_json::Value::Null);
        }
    });

    // Register disconnect handler for subscription cleanup
    ws_builder.handle_disconnect(|conn| {
        conn.cancel_all_subscriptions();
    });

    // Register handlers
    handlers::auth::register(&mut ws_builder, state.clone());
    handlers::settings::register(&mut ws_builder, state.clone());
    handlers::stack::register(&mut ws_builder, state.clone());
    handlers::docker::register(&mut ws_builder, state.clone());
    handlers::service::register(&mut ws_builder, state.clone());
    handlers::terminal::register(&mut ws_builder, state.clone());
    handlers::terminal::register_binary_handler(&mut ws_builder, state.clone());
    handlers::image_updates::register(&mut ws_builder, state.clone());

    let ws_server = Arc::new(ws_builder);

    // Spawn the control loop for WsServer (handles disconnect_others, etc.)
    ws_server.spawn_control_loop(ws_control_rx);

    // Build router
    let mut app = Router::new()
        .route("/healthz", get({
            let state_healthz = state.clone();
            move || {
                let state = state_healthz.clone();
                async move {
                    if state.event_watcher_ready.load(Ordering::Relaxed) {
                        (StatusCode::OK, "ok")
                    } else {
                        (StatusCode::SERVICE_UNAVAILABLE, "starting")
                    }
                }
            }
        }))
        .route("/ws", get({
            let ws_server = ws_server.clone();
            move |ws_upgrade: WebSocketUpgrade| {
                let ws_server = ws_server.clone();
                async move {
                    ws_upgrade
                        .max_frame_size(1 << 20)
                        .on_upgrade(move |socket| async move {
                            ws_server.accept(socket);
                        })
                }
            }
        }));

    // Dev mode endpoints
    if config.dev {
        let state_reset = state.clone();
        app = app.route("/api/dev/reset-db", post(move || {
            let state = state_reset.clone();
            async move {
                if let Err(e) = state.users.delete_all() {
                    error!("dev reset-db: delete users: {e}");
                    return (StatusCode::INTERNAL_SERVER_ERROR, format!("{e}"));
                }
                if let Err(e) = state.users.create("admin", "testpass123") {
                    error!("dev reset-db: create admin: {e}");
                    return (StatusCode::INTERNAL_SERVER_ERROR, format!("{e}"));
                }
                state.need_setup.store(false, Ordering::Relaxed);
                state.login_limiter.reset_all();
                (StatusCode::OK, "ok".to_string())
            }
        }));

        app = app.route("/api/mock/reset", post(move || async move {
            match reset_via_daemon().await {
                Ok(_) => (StatusCode::OK, "ok".to_string()),
                Err(e) => {
                    error!("mock reset proxy: {e}");
                    (StatusCode::BAD_GATEWAY, format!("{e}"))
                }
            }
        }));

        // Memory stats endpoints for performance benchmarks
        let mem_tracker = Arc::new(debug::MemTracker::new());
        debug::MemTracker::spawn_sampler(mem_tracker.clone());

        let mt_get = mem_tracker.clone();
        app = app.route("/api/debug/memstats", get(move || {
            let mt = mt_get.clone();
            async move { Json(mt.stats()) }
        }));

        let mt_reset = mem_tracker;
        app = app.route("/api/debug/memstats/reset", post(move || {
            let mt = mt_reset.clone();
            async move {
                mt.reset();
                Json(serde_json::json!({"ok": true}))
            }
        }));
    }

    let app = app
        .fallback(embed::spa_handler)
        .layer(CompressionLayer::new().gzip(true));

    // Start background tasks
    let shutdown_token = tokio_util::sync::CancellationToken::new();
    broadcast::watcher::spawn(state.clone(), dispatch_rx, shutdown_token.clone());
    handlers::image_updates::spawn_checker(state.clone(), shutdown_token.clone());
    compose::watcher::spawn(
        config.stacks_dir.clone(),
        state.dispatch_tx.clone(),
        shutdown_token.clone(),
    );

    // Bind and serve
    let addr = SocketAddr::from(([0, 0, 0, 0], config.port));
    let listener = TcpListener::bind(addr).await.expect("failed to bind");
    info!("listening on {addr}");

    axum::serve(listener, app)
        .with_graceful_shutdown(shutdown_signal())
        .await
        .expect("server error");

    shutdown_token.cancel();
    info!("shutdown complete");
}

/// Proxy POST /_mock/reset to the mock daemon via DOCKER_HOST unix socket.
async fn reset_via_daemon() -> Result<(), Box<dyn std::error::Error>> {
    let dh = std::env::var("DOCKER_HOST").unwrap_or_default();
    if !dh.starts_with("unix://") {
        return Err(format!("DOCKER_HOST is not a Unix socket (got {dh:?})").into());
    }
    let sock_path = dh.trim_start_matches("unix://");

    let stream = tokio::net::UnixStream::connect(sock_path).await?;
    let (mut sender, conn) = hyper::client::conn::http1::handshake(
        hyper_util::rt::TokioIo::new(stream),
    ).await?;
    tokio::spawn(async move { let _ = conn.await; });

    let req = hyper::Request::builder()
        .method("POST")
        .uri("/_mock/reset")
        .header("host", "docker")
        .body(http_body_util::Empty::<bytes::Bytes>::new())?;

    let resp = sender.send_request(req).await?;
    if !resp.status().is_success() {
        return Err(format!("/_mock/reset returned {}", resp.status()).into());
    }

    Ok(())
}

fn get_or_create_setting(db: &redb::Database, key: &str) -> Option<String> {
    let read_txn = db.begin_read().ok()?;
    let table = read_txn.open_table(db::SETTINGS_TABLE).ok()?;
    table.get(key).ok()?.map(|v| v.value().to_string())
}

fn set_setting(db: &redb::Database, key: &str, value: &str) {
    let write_txn = db.begin_write().expect("failed to begin write txn for setting");
    {
        let mut table = write_txn
            .open_table(db::SETTINGS_TABLE)
            .expect("failed to open settings table");
        table.insert(key, value).expect("failed to insert setting");
    }
    write_txn.commit().expect("failed to commit setting");
}

async fn shutdown_signal() {
    let ctrl_c = async {
        signal::ctrl_c().await.expect("failed to listen for ctrl+c");
    };

    #[cfg(unix)]
    let terminate = async {
        signal::unix::signal(signal::unix::SignalKind::terminate())
            .expect("failed to listen for SIGTERM")
            .recv()
            .await;
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        () = ctrl_c => info!("received SIGINT"),
        () = terminate => info!("received SIGTERM"),
    }
}
