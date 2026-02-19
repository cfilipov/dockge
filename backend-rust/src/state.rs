use crate::config::Config;
use serde::{Deserialize, Serialize};
use socketioxide::SocketIo;
use sqlx::SqlitePool;
use std::collections::HashMap;
use std::path::PathBuf;
use std::sync::Arc;
use tokio::sync::RwLock;

/// Per-stack simple info for the stack list (hot path)
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SimpleStackInfo {
    pub name: String,
    pub status: i32,
    pub started: bool,
    pub recreate_necessary: bool,
    pub image_updates_available: bool,
    pub tags: Vec<String>,
    pub is_managed_by_dockge: bool,
    pub compose_file_name: String,
    pub compose_override_file_name: String,
    pub endpoint: String,
}

/// Image update cache entry
#[derive(Debug, Clone)]
pub struct StackUpdateInfo {
    pub has_updates: bool,
    pub services: HashMap<String, bool>,
}

/// Terminal handle for tracking active terminals
pub struct TerminalHandle {
    pub name: String,
    pub child: Option<tokio::process::Child>,
    pub kill_tx: Option<tokio::sync::oneshot::Sender<()>>,
    pub buffer: Vec<String>,
    pub subscribers: Vec<String>, // socket IDs
}

/// Shared application state
pub struct AppState {
    pub db: SqlitePool,
    pub config: Config,
    pub stacks_dir: PathBuf,
    pub data_dir: PathBuf,
    pub stack_cache: RwLock<HashMap<String, SimpleStackInfo>>,
    pub update_cache: RwLock<HashMap<String, StackUpdateInfo>>,
    pub recreate_cache: RwLock<HashMap<String, bool>>,
    pub terminals: RwLock<HashMap<String, TerminalHandle>>,
    pub jwt_secret: RwLock<String>,
    pub need_setup: RwLock<bool>,
    pub io: SocketIo,
    pub version: String,
    pub latest_version: RwLock<Option<String>>,
}

impl AppState {
    pub async fn new(
        db: SqlitePool,
        config: Config,
        io: SocketIo,
    ) -> Arc<Self> {
        let stacks_dir = config.stacks_dir.clone();
        let data_dir = config.data_dir.clone();

        Arc::new(Self {
            db,
            config,
            stacks_dir,
            data_dir,
            stack_cache: RwLock::new(HashMap::new()),
            update_cache: RwLock::new(HashMap::new()),
            recreate_cache: RwLock::new(HashMap::new()),
            terminals: RwLock::new(HashMap::new()),
            jwt_secret: RwLock::new(String::new()),
            need_setup: RwLock::new(false),
            io,
            version: "1.5.0".to_string(),
            latest_version: RwLock::new(None),
        })
    }
}
