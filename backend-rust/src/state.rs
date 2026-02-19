use crate::config::Config;
use serde::{Deserialize, Serialize};
use socketioxide::SocketIo;
use sqlx::SqlitePool;
use std::collections::HashMap;
use std::path::PathBuf;
use std::sync::Arc;
use std::time::Duration;
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

/// Shared application state
#[allow(dead_code)]
pub struct AppState {
    pub db: SqlitePool,
    pub config: Config,
    pub stacks_dir: PathBuf,
    pub data_dir: PathBuf,
    pub stack_cache: RwLock<HashMap<String, SimpleStackInfo>>,
    pub update_cache: RwLock<HashMap<String, StackUpdateInfo>>,
    pub recreate_cache: RwLock<HashMap<String, bool>>,
    pub jwt_secret: RwLock<String>,
    pub need_setup: RwLock<bool>,
    pub io: SocketIo,
    pub http_client: reqwest::Client,
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

        let http_client = reqwest::Client::builder()
            .timeout(Duration::from_secs(15))
            .pool_max_idle_per_host(2)
            .build()
            .expect("Failed to build HTTP client");

        Arc::new(Self {
            db,
            config,
            stacks_dir,
            data_dir,
            stack_cache: RwLock::new(HashMap::new()),
            update_cache: RwLock::new(HashMap::new()),
            recreate_cache: RwLock::new(HashMap::new()),
            jwt_secret: RwLock::new(String::new()),
            need_setup: RwLock::new(false),
            io,
            http_client,
            version: "1.5.0".to_string(),
            latest_version: RwLock::new(None),
        })
    }
}
