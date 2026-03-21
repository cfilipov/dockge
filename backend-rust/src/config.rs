use clap::Parser;

#[derive(Parser, Debug)]
#[command(name = "dockge-rust", about = "Docker Compose stack manager")]
pub struct Cli {
    /// HTTP server port
    #[arg(long, default_value_t = 5001, env = "DOCKGE_PORT")]
    pub port: u16,

    /// Path to stacks directory
    #[arg(long, default_value = "/opt/stacks", env = "DOCKGE_STACKS_DIR")]
    pub stacks_dir: String,

    /// Path to data directory (database)
    #[arg(long, default_value = "./data", env = "DOCKGE_DATA_DIR")]
    pub data_dir: String,

    /// Development mode
    #[arg(long, default_value_t = false)]
    pub dev: bool,

    /// Log level: debug, info, warn, error
    #[arg(long, default_value = "info", env = "DOCKGE_LOG_LEVEL")]
    pub log_level: String,

    /// Disable authentication
    #[arg(long, default_value_t = false, env = "DOCKGE_NO_AUTH")]
    pub no_auth: bool,

    /// Max worker threads (0 = use all CPUs)
    #[arg(long, default_value_t = 1, env = "DOCKGE_MAX_PROCS")]
    pub max_procs: usize,
}

#[derive(Debug, Clone)]
pub struct Config {
    pub port: u16,
    pub stacks_dir: String,
    pub data_dir: String,
    pub dev: bool,
    pub log_level: String,
    pub no_auth: bool,
    #[allow(dead_code)]
    pub max_procs: usize,
}

impl From<&Cli> for Config {
    fn from(cli: &Cli) -> Self {
        Config {
            port: cli.port,
            stacks_dir: cli.stacks_dir.clone(),
            data_dir: cli.data_dir.clone(),
            dev: cli.dev,
            log_level: cli.log_level.clone(),
            no_auth: cli.no_auth,
            max_procs: cli.max_procs,
        }
    }
}
