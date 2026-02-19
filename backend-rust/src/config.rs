use clap::Parser;
use std::path::PathBuf;

#[derive(Parser, Debug, Clone)]
#[command(name = "dockge-backend", about = "Dockge backend server (Rust)")]
pub struct Config {
    #[arg(long, env = "DOCKGE_PORT", default_value = "5001")]
    pub port: u16,

    #[arg(long, env = "DOCKGE_HOSTNAME")]
    pub hostname: Option<String>,

    #[arg(long, env = "DOCKGE_DATA_DIR", default_value = "./data/")]
    pub data_dir: PathBuf,

    #[arg(long, env = "DOCKGE_STACKS_DIR", default_value = "/opt/stacks")]
    pub stacks_dir: PathBuf,

    #[arg(long, env = "DOCKGE_SSL_KEY")]
    pub ssl_key: Option<String>,

    #[arg(long, env = "DOCKGE_SSL_CERT")]
    pub ssl_cert: Option<String>,

    #[arg(long, env = "DOCKGE_SSL_KEY_PASSPHRASE")]
    pub ssl_key_passphrase: Option<String>,

    #[arg(long, env = "DOCKGE_ENABLE_CONSOLE", default_value = "false")]
    pub enable_console: bool,
}

impl Config {
    pub fn db_path(&self) -> PathBuf {
        self.data_dir.join("dockge.db")
    }

    pub fn is_dev(&self) -> bool {
        std::env::var("NODE_ENV")
            .map(|v| v == "development")
            .unwrap_or(false)
    }
}
