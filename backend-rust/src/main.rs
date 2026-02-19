mod config;
mod db;
mod docker;
mod error;
mod handlers;
mod models;
pub mod socket_args;
mod state;
mod terminal;
mod update_checker;

use axum::Router;
use clap::Parser;
use config::Config;
use models::settings as settings_model;
use models::user::{self, User};
use socketioxide::extract::SocketRef;
use socketioxide::SocketIo;
use state::AppState;
use tower_http::compression::CompressionLayer;
use tower_http::cors::CorsLayer;
use tower_http::services::{ServeDir, ServeFile};
use tracing::{info, warn};

fn main() {
    let runtime = tokio::runtime::Builder::new_multi_thread()
        .worker_threads(2)
        .max_blocking_threads(4)
        .enable_all()
        .build()
        .expect("Failed to build tokio runtime");

    runtime.block_on(async_main());
}

async fn async_main() {
    // Initialize tracing
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "dockge_backend=info,tower_http=info".into()),
        )
        .init();

    let config = Config::parse();
    info!("Dockge Rust backend starting...");
    info!("Data dir: {}", config.data_dir.display());
    info!("Stacks dir: {}", config.stacks_dir.display());
    info!("Port: {}", config.port);

    // Ensure directories exist
    std::fs::create_dir_all(&config.data_dir).expect("Failed to create data directory");
    std::fs::create_dir_all(&config.stacks_dir).expect("Failed to create stacks directory");

    // Initialize database
    let db_path = config.db_path();
    let pool = db::init_pool(&db_path)
        .await
        .expect("Failed to initialize database");
    db::run_migrations(&pool)
        .await
        .expect("Failed to run database migrations");

    // Initialize Socket.IO
    let (sio_layer, io) = SocketIo::builder()
        .max_buffer_size(128 * 1024) // 128KB
        .build_layer();

    // Create app state
    let state = AppState::new(pool.clone(), config.clone(), io.clone()).await;

    // Initialize JWT secret
    {
        let existing_secret: Option<(Option<String>,)> = sqlx::query_as(
            "SELECT value FROM setting WHERE key = 'jwtSecret'"
        )
        .fetch_optional(&state.db)
        .await
        .unwrap_or(None);

        let secret = if let Some((Some(secret_val),)) = existing_secret {
            info!("Loaded JWT secret from database");
            // The value is stored JSON-encoded, try to parse it
            serde_json::from_str::<String>(&secret_val).unwrap_or(secret_val)
        } else {
            let new_secret = user::generate_password_hash(&user::gen_secret(64));
            let secret_json = serde_json::to_string(&new_secret).unwrap();
            sqlx::query("INSERT INTO setting (key, value) VALUES ('jwtSecret', ?)")
                .bind(&secret_json)
                .execute(&state.db)
                .await
                .ok();
            info!("Generated and stored new JWT secret");
            new_secret
        };

        let mut jwt = state.jwt_secret.write().await;
        *jwt = secret;
    }

    // Check if setup is needed
    {
        let user_count = User::count(&state.db).await.unwrap_or(0);
        if user_count == 0 {
            info!("No users found, setup required");
            let mut need_setup = state.need_setup.write().await;
            *need_setup = true;
        }
    }

    // Set up Socket.IO connection handler
    let state_for_io = state.clone();
    io.ns("/", move |socket: SocketRef| {
        let state = state_for_io.clone();

        async move {
            let endpoint = socket
                .req_parts()
                .headers
                .get("endpoint")
                .and_then(|v| v.to_str().ok())
                .unwrap_or("")
                .to_string();

            socket.extensions.insert(handlers::auth::SocketEndpoint(endpoint.clone()));

            if endpoint.is_empty() {
                info!("Socket connected (direct): {}", socket.id);
            } else {
                info!("Socket connected (agent), endpoint: {}", endpoint);
            }

            // Send info on connect
            handlers::auth::send_info(&state, &socket, true).await;

            // Check if setup needed
            {
                let need_setup = state.need_setup.read().await;
                if *need_setup {
                    info!("Redirecting to setup page");
                    socket.emit("setup", &()).ok();
                }
            }

            // Check auto login
            let disable_auth = settings_model::get(&state.db, "disableAuth")
                .await
                .ok()
                .flatten()
                .and_then(|v| v.as_bool())
                .unwrap_or(false);

            if disable_auth {
                info!("Disabled Auth: auto login to first user");
                if let Ok(Some(user)) = User::find_first(&state.db).await {
                    socket.extensions.insert(handlers::auth::SocketUserId(user.id));
                    socket.extensions.insert(handlers::auth::SocketUsername(user.username.clone()));
                    handlers::auth::after_login(&state, &socket).await;
                    socket.emit("autoLogin", &()).ok();
                }
            }

            // Register all event handlers

            // Auth handlers (direct, not via agent proxy)
            handlers::auth::register(&socket, state.clone());

            // Settings handlers (direct)
            handlers::settings::register(&socket, state.clone());

            // Agent management handlers (direct)
            handlers::agent::register(&socket, state.clone());

            // Stack handlers (registered directly — the agent proxy will route to them)
            handlers::stack::register_agent_handlers(&socket, state.clone());

            // Terminal handlers
            handlers::terminal::register_agent_handlers(&socket, state.clone());

            // Register the agent proxy (routes "agent" events to the above handlers)
            handlers::agent::register_agent_proxy(&socket, state.clone());

            // Handle disconnect
            socket.on_disconnect(|socket: SocketRef| async move {
                info!("Socket disconnected: {}", socket.id);
            });
        }
    });

    // Start background tasks
    let state_for_cron = state.clone();
    tokio::spawn(async move {
        // Stack list refresh every 10 seconds
        let mut interval = tokio::time::interval(std::time::Duration::from_secs(10));
        loop {
            interval.tick().await;
            handlers::stack::broadcast_stack_list(&state_for_cron).await;
        }
    });

    // Start image update checker background task
    update_checker::start_background_checker(state.clone());

    // Build Axum app
    let frontend_dist = std::path::Path::new("frontend-dist");
    let app = if frontend_dist.exists() {
        info!("Serving static files from frontend-dist/");
        let serve_dir = ServeDir::new("frontend-dist")
            .not_found_service(ServeFile::new("frontend-dist/index.html"));

        Router::new()
            .fallback_service(serve_dir)
            .layer(sio_layer)
            .layer(CompressionLayer::new())
    } else {
        warn!("frontend-dist/ not found, only Socket.IO will be served");
        Router::new()
            .layer(sio_layer)
            .layer(CompressionLayer::new())
    };

    // Add CORS for development
    let app = if config.is_dev() {
        app.layer(CorsLayer::permissive())
    } else {
        app
    };

    // Bind and serve
    let bind_addr = if let Some(ref hostname) = config.hostname {
        format!("{}:{}", hostname, config.port)
    } else {
        format!("0.0.0.0:{}", config.port)
    };

    let listener = tokio::net::TcpListener::bind(&bind_addr)
        .await
        .unwrap_or_else(|_| panic!("Failed to bind to {}", bind_addr));

    info!("Listening on {}", bind_addr);

    axum::serve(listener, app)
        .with_graceful_shutdown(shutdown_signal())
        .await
        .expect("Server error");

    info!("Server shut down gracefully");
}

async fn shutdown_signal() {
    let ctrl_c = async {
        tokio::signal::ctrl_c()
            .await
            .expect("Failed to install Ctrl+C handler");
    };

    #[cfg(unix)]
    let terminate = async {
        tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())
            .expect("Failed to install signal handler")
            .recv()
            .await;
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => info!("Received Ctrl+C, shutting down..."),
        _ = terminate => info!("Received SIGTERM, shutting down..."),
    }
}
