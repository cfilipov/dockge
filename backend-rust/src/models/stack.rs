use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::collections::HashMap;
use std::path::{Path, PathBuf};
use tokio::fs;
use tracing::warn;

use crate::error::{AppError, AppResult};
use crate::state::SimpleStackInfo;

// Status constants (matching common/util-common.ts)
pub const UNKNOWN: i32 = 0;
pub const CREATED_FILE: i32 = 1;
pub const CREATED_STACK: i32 = 2;
pub const RUNNING: i32 = 3;
pub const EXITED: i32 = 4;
pub const RUNNING_AND_EXITED: i32 = 5;
pub const UNHEALTHY: i32 = 6;

pub const ACCEPTED_COMPOSE_FILE_NAMES: &[&str] = &[
    "compose.yaml",
    "docker-compose.yaml",
    "docker-compose.yml",
    "compose.yml",
];

pub const ACCEPTED_COMPOSE_OVERRIDE_FILE_NAMES: &[&str] = &[
    "compose.override.yaml",
    "compose.override.yml",
    "docker-compose.override.yaml",
    "docker-compose.override.yml",
];

#[derive(Debug, Clone)]
pub struct Stack {
    pub name: String,
    pub status: i32,
    pub compose_yaml: String,
    pub compose_env: String,
    pub compose_override_yaml: String,
    pub compose_file_name: String,
    pub compose_override_file_name: String,
    pub config_file_path: Option<String>,
    pub stacks_dir: PathBuf,
}

/// JSON output from `docker compose ls --format json`
#[derive(Debug, Deserialize)]
#[allow(dead_code)]
pub struct ComposeLsEntry {
    #[serde(alias = "Name")]
    pub name: String,
    #[serde(alias = "Status")]
    pub status: String,
    #[serde(alias = "ConfigFiles", default)]
    pub config_files: Option<String>,
}

/// JSON output from `docker compose ps --format json`
#[derive(Debug, Deserialize, Serialize, Clone)]
#[allow(dead_code)]
pub struct ComposePsEntry {
    #[serde(alias = "Service", default)]
    pub service: String,
    #[serde(alias = "State", default)]
    pub state: String,
    #[serde(alias = "Name", default)]
    pub name: String,
    #[serde(alias = "Health", default)]
    pub health: String,
    #[serde(alias = "Image", default)]
    pub image: String,
}

impl Stack {
    pub fn new(name: String, stacks_dir: PathBuf) -> Self {
        Self {
            name,
            status: UNKNOWN,
            compose_yaml: String::new(),
            compose_env: String::new(),
            compose_override_yaml: String::new(),
            compose_file_name: "compose.yaml".to_string(),
            compose_override_file_name: "compose.override.yaml".to_string(),
            config_file_path: None,
            stacks_dir,
        }
    }

    pub fn path(&self) -> PathBuf {
        self.stacks_dir.join(&self.name)
    }

    pub fn full_path(&self) -> PathBuf {
        let p = self.path();
        if p.is_absolute() {
            p
        } else {
            std::env::current_dir().unwrap_or_default().join(p)
        }
    }

    pub fn is_managed_by_dockge(&self) -> bool {
        let p = self.path();
        p.exists() && p.is_dir()
    }

    pub fn is_started(&self) -> bool {
        self.status == RUNNING || self.status == RUNNING_AND_EXITED || self.status == UNHEALTHY
    }

    /// Load compose files from disk
    pub async fn load_from_disk(&mut self) -> AppResult<()> {
        let dir = self.path();

        // Find the compose file
        for filename in ACCEPTED_COMPOSE_FILE_NAMES {
            let p = dir.join(filename);
            if p.exists() {
                self.compose_file_name = filename.to_string();
                self.compose_yaml = fs::read_to_string(&p).await.unwrap_or_default();
                break;
            }
        }

        // Find the override file
        for filename in ACCEPTED_COMPOSE_OVERRIDE_FILE_NAMES {
            let p = dir.join(filename);
            if p.exists() {
                self.compose_override_file_name = filename.to_string();
                self.compose_override_yaml = fs::read_to_string(&p).await.unwrap_or_default();
                break;
            }
        }

        // Load .env
        let env_path = dir.join(".env");
        if env_path.exists() {
            self.compose_env = fs::read_to_string(&env_path).await.unwrap_or_default();
        }

        Ok(())
    }

    /// Convert to simple JSON for stack list
    pub fn to_simple_json(&self, endpoint: &str, recreate_necessary: bool, image_updates_available: bool) -> SimpleStackInfo {
        SimpleStackInfo {
            name: self.name.clone(),
            status: self.status,
            started: self.is_started(),
            recreate_necessary,
            image_updates_available,
            tags: vec![],
            is_managed_by_dockge: self.is_managed_by_dockge(),
            compose_file_name: self.compose_file_name.clone(),
            compose_override_file_name: self.compose_override_file_name.clone(),
            endpoint: endpoint.to_string(),
        }
    }

    /// Convert to full JSON for getStack response
    pub async fn to_json(&self, endpoint: &str, recreate_necessary: bool, image_updates_available: bool, primary_hostname: &str) -> Value {
        serde_json::json!({
            "name": self.name,
            "status": self.status,
            "started": self.is_started(),
            "recreateNecessary": recreate_necessary,
            "imageUpdatesAvailable": image_updates_available,
            "tags": [],
            "isManagedByDockge": self.is_managed_by_dockge(),
            "composeFileName": self.compose_file_name,
            "composeOverrideFileName": self.compose_override_file_name,
            "endpoint": endpoint,
            "composeYAML": self.compose_yaml,
            "composeENV": self.compose_env,
            "composeOverrideYAML": self.compose_override_yaml,
            "primaryHostname": primary_hostname,
        })
    }

    /// Validate the stack
    pub fn validate(&self) -> AppResult<()> {
        // Check name: [a-z0-9_-] only
        let re = regex::Regex::new(r"^[a-z0-9_-]+$").unwrap();
        if !re.is_match(&self.name) {
            return Err(AppError::Validation(
                "Stack name can only contain [a-z][0-9] _ - only".to_string()
            ));
        }

        // Check YAML syntax
        let _: Value = serde_yaml::from_str(&self.compose_yaml)
            .map_err(|e| AppError::Validation(format!("Invalid YAML: {}", e)))?;

        // Check override YAML if present
        if !self.compose_override_yaml.trim().is_empty() {
            let _: Value = serde_yaml::from_str(&self.compose_override_yaml)
                .map_err(|e| AppError::Validation(format!("Invalid override YAML: {}", e)))?;
        }

        // Check .env format
        let lines: Vec<&str> = self.compose_env.lines().collect();
        if lines.len() == 1 && !lines[0].contains('=') && !lines[0].is_empty() {
            return Err(AppError::Validation("Invalid .env format".to_string()));
        }

        Ok(())
    }

    /// Save compose files to disk
    pub async fn save(&self, is_add: bool) -> AppResult<()> {
        self.validate()?;

        let dir = self.path();

        if is_add {
            if dir.exists() {
                return Err(AppError::Validation("Stack name already exists".to_string()));
            }
            fs::create_dir_all(&dir).await?;
        } else if !dir.exists() {
            return Err(AppError::NotFound("Stack not found".to_string()));
        }

        // Write compose file
        fs::write(dir.join(&self.compose_file_name), &self.compose_yaml).await?;

        // Write .env
        let env_path = dir.join(".env");
        if env_path.exists() || !self.compose_env.trim().is_empty() {
            fs::write(&env_path, &self.compose_env).await?;
        }

        // Write override file
        let override_path = dir.join(&self.compose_override_file_name);
        if override_path.exists() || !self.compose_override_yaml.trim().is_empty() {
            fs::write(&override_path, &self.compose_override_yaml).await?;
        }

        Ok(())
    }

    /// Get compose CLI options (handles global.env)
    pub fn get_compose_options(&self, command: &str, extra_args: &[&str]) -> Vec<String> {
        let mut options = vec!["compose".to_string()];

        let global_env_path = self.stacks_dir.join("global.env");
        if global_env_path.exists() {
            options.push("--env-file".to_string());
            options.push("../global.env".to_string());

            let local_env = self.path().join(".env");
            if local_env.exists() {
                options.push("--env-file".to_string());
                options.push("./.env".to_string());
            }
        }

        options.push(command.to_string());
        for arg in extra_args {
            options.push(arg.to_string());
        }

        options
    }

    /// Get the stack list by scanning the stacks directory and running docker compose ls
    pub async fn get_stack_list(stacks_dir: &Path) -> AppResult<HashMap<String, Stack>> {
        let mut stack_list = HashMap::new();

        // Scan stacks directory for managed stacks
        if let Ok(mut entries) = fs::read_dir(stacks_dir).await {
            while let Ok(Some(entry)) = entries.next_entry().await {
                let name = entry.file_name().to_string_lossy().to_string();
                if let Ok(ft) = entry.file_type().await {
                    if !ft.is_dir() {
                        continue;
                    }
                }

                // Check if any compose file exists
                let dir = stacks_dir.join(&name);
                let has_compose = ACCEPTED_COMPOSE_FILE_NAMES.iter().any(|f| dir.join(f).exists());
                if !has_compose {
                    continue;
                }

                let mut stack = Stack::new(name.clone(), stacks_dir.to_path_buf());
                stack.status = CREATED_FILE;
                if let Err(e) = stack.load_from_disk().await {
                    warn!("Failed to load stack {}: {}", name, e);
                    continue;
                }
                stack_list.insert(name, stack);
            }
        }

        // Get status from docker compose ls
        match get_compose_ls_status().await {
            Ok(compose_list) => {
                for entry in compose_list {
                    let stack = stack_list.entry(entry.name.clone())
                        .or_insert_with(|| Stack::new(entry.name.clone(), stacks_dir.to_path_buf()));

                    // Skip the "dockge" stack if not managed
                    if entry.name == "dockge" && !stack.is_managed_by_dockge() {
                        stack_list.remove(&entry.name);
                        continue;
                    }

                    stack.status = status_convert(&entry.status);
                    stack.config_file_path = entry.config_files;
                }
            }
            Err(e) => {
                warn!("docker compose ls failed: {}", e);
            }
        }

        Ok(stack_list)
    }

    /// Get a single stack by name
    pub async fn get_stack(stacks_dir: &Path, name: &str) -> AppResult<Stack> {
        let dir = stacks_dir.join(name);

        if dir.exists() && dir.is_dir() {
            let mut stack = Stack::new(name.to_string(), stacks_dir.to_path_buf());
            stack.load_from_disk().await?;
            Ok(stack)
        } else {
            // Try to find it in the compose ls output (unmanaged stack)
            let stack_list = Self::get_stack_list(stacks_dir).await?;
            stack_list.get(name)
                .cloned()
                .ok_or_else(|| AppError::NotFound("Stack not found".to_string()))
        }
    }

    /// Get per-service status by running docker compose ps
    pub async fn get_service_status_list(&self) -> AppResult<HashMap<String, Vec<Value>>> {
        let mut status_list: HashMap<String, Vec<Value>> = HashMap::new();

        let args = self.get_compose_options("ps", &["--format", "json"]);
        let output = tokio::process::Command::new("docker")
            .args(&args)
            .current_dir(self.path())
            .output()
            .await?;

        if !output.stdout.is_empty() {
            let stdout = String::from_utf8_lossy(&output.stdout);
            for line in stdout.lines() {
                if line.trim().is_empty() {
                    continue;
                }
                // Try parsing as array or single object
                if let Ok(entries) = serde_json::from_str::<Vec<ComposePsEntry>>(line) {
                    for entry in entries {
                        add_ps_entry(&mut status_list, entry);
                    }
                } else if let Ok(entry) = serde_json::from_str::<ComposePsEntry>(line) {
                    add_ps_entry(&mut status_list, entry);
                }
            }
        }

        Ok(status_list)
    }
}

fn add_ps_entry(status_list: &mut HashMap<String, Vec<Value>>, entry: ComposePsEntry) {
    let service_name = entry.service.clone();
    let status = if entry.health.is_empty() {
        entry.state.clone()
    } else {
        entry.health.clone()
    };

    let value = serde_json::json!({
        "status": status,
        "name": entry.name,
        "image": entry.image,
    });

    status_list.entry(service_name).or_default().push(value);
}

/// Run `docker compose ls --all --format json` and parse output
async fn get_compose_ls_status() -> AppResult<Vec<ComposeLsEntry>> {
    let output = tokio::process::Command::new("docker")
        .args(["compose", "ls", "--all", "--format", "json"])
        .output()
        .await?;

    if output.stdout.is_empty() {
        return Ok(vec![]);
    }

    let stdout = String::from_utf8_lossy(&output.stdout);
    let list: Vec<ComposeLsEntry> = serde_json::from_str(&stdout)
        .map_err(|e| AppError::Internal(format!("Failed to parse compose ls: {} (output: {})", e, stdout)))?;

    Ok(list)
}

/// Convert docker compose ls status string to our status code
pub fn status_convert(status: &str) -> i32 {
    if status.starts_with("created") {
        return CREATED_STACK;
    }

    let running_count = regex::Regex::new(r"running\((\d+)\)")
        .unwrap()
        .captures(status)
        .and_then(|c| c.get(1))
        .and_then(|m| m.as_str().parse::<i32>().ok())
        .unwrap_or(0);

    let exited_count = regex::Regex::new(r"exited\((\d+)\)")
        .unwrap()
        .captures(status)
        .and_then(|c| c.get(1))
        .and_then(|m| m.as_str().parse::<i32>().ok())
        .unwrap_or(0);

    if running_count > 0 && exited_count > 0 {
        RUNNING_AND_EXITED
    } else if running_count > 0 {
        RUNNING
    } else if exited_count > 0 {
        EXITED
    } else if status.contains("running") {
        RUNNING
    } else if status.contains("exited") {
        EXITED
    } else {
        UNKNOWN
    }
}

/// Validate using docker compose config --dry-run
pub async fn validate_with_docker(
    compose_yaml: &str,
    compose_env: &str,
    compose_file_name: &str,
) -> AppResult<()> {
    let temp_dir = tempfile::tempdir()
        .map_err(|e| AppError::Internal(format!("Failed to create temp dir: {}", e)))?;

    let yaml_path = temp_dir.path().join(compose_file_name);
    fs::write(&yaml_path, compose_yaml).await?;

    let has_env = !compose_env.trim().is_empty();
    if has_env {
        let env_path = temp_dir.path().join(".env");
        fs::write(&env_path, compose_env).await?;
    }

    let mut args = vec!["compose", "-f", compose_file_name];
    if has_env {
        args.extend_from_slice(&["--env-file", ".env"]);
    }
    args.extend_from_slice(&["config", "--dry-run"]);

    let output = tokio::process::Command::new("docker")
        .args(&args)
        .current_dir(temp_dir.path())
        .output()
        .await?;

    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        let mut msg = stderr.trim().to_string();
        // Remove the "validating /tmp/..." prefix
        if let Some(idx) = msg.find(": ") {
            if msg.starts_with("validating ") {
                msg = msg[idx + 2..].to_string();
            }
        }
        if msg.is_empty() {
            msg = "Docker compose validation failed".to_string();
        }
        return Err(AppError::Validation(msg));
    }

    Ok(())
}
