use serde_json::Value;
use std::collections::HashMap;
use std::path::Path;

use crate::error::{AppError, AppResult};

/// Run a docker compose command and return the exit code.
/// Output is streamed to the provided callback.
pub async fn compose_exec(
    stack_dir: &Path,
    args: &[String],
    on_output: impl Fn(&str) + Send + Sync + 'static,
) -> AppResult<i32> {
    use tokio::io::{AsyncBufReadExt, BufReader};
    use tokio::process::Command;

    let mut cmd = Command::new("docker");
    cmd.args(args)
        .current_dir(stack_dir)
        .stdout(std::process::Stdio::piped())
        .stderr(std::process::Stdio::piped());

    let mut child = cmd.spawn()?;

    let stdout = child.stdout.take();
    let stderr = child.stderr.take();

    let on_output2 = std::sync::Arc::new(on_output);
    let on_output3 = on_output2.clone();

    let stdout_task = tokio::spawn(async move {
        if let Some(stdout) = stdout {
            let reader = BufReader::new(stdout);
            let mut lines = reader.lines();
            while let Ok(Some(line)) = lines.next_line().await {
                on_output2(&format!("{}\r\n", line));
            }
        }
    });

    let stderr_task = tokio::spawn(async move {
        if let Some(stderr) = stderr {
            let reader = BufReader::new(stderr);
            let mut lines = reader.lines();
            while let Ok(Some(line)) = lines.next_line().await {
                on_output3(&format!("{}\r\n", line));
            }
        }
    });

    let _ = tokio::join!(stdout_task, stderr_task);

    let status = child.wait().await?;
    Ok(status.code().unwrap_or(-1))
}

/// Run a docker compose command and capture all output (no streaming)
pub async fn compose_exec_capture(
    stack_dir: &Path,
    args: &[String],
) -> AppResult<(i32, String, String)> {
    let output = tokio::process::Command::new("docker")
        .args(args)
        .current_dir(stack_dir)
        .output()
        .await?;

    let stdout = String::from_utf8_lossy(&output.stdout).to_string();
    let stderr = String::from_utf8_lossy(&output.stderr).to_string();
    let code = output.status.code().unwrap_or(-1);

    Ok((code, stdout, stderr))
}

/// Get docker network list
pub async fn get_network_list() -> AppResult<Vec<String>> {
    let output = tokio::process::Command::new("docker")
        .args(["network", "ls", "--format", "{{.Name}}"])
        .output()
        .await?;

    let stdout = String::from_utf8_lossy(&output.stdout);
    let mut list: Vec<String> = stdout
        .lines()
        .filter(|l| !l.is_empty())
        .map(|l| l.to_string())
        .collect();
    list.sort();

    Ok(list)
}

/// Get docker stats for all containers
pub async fn get_docker_stats() -> AppResult<HashMap<String, Value>> {
    let mut stats = HashMap::new();

    let output = tokio::process::Command::new("docker")
        .args(["stats", "--format", "json", "--no-stream"])
        .output()
        .await?;

    if output.stdout.is_empty() {
        return Ok(stats);
    }

    let stdout = String::from_utf8_lossy(&output.stdout);
    for line in stdout.lines() {
        if let Ok(obj) = serde_json::from_str::<Value>(line) {
            if let Some(name) = obj.get("Name").and_then(|v| v.as_str()) {
                stats.insert(name.to_string(), obj);
            }
        }
    }

    Ok(stats)
}

/// Get docker inspect for a container
pub async fn container_inspect(container_name: &str) -> AppResult<String> {
    let output = tokio::process::Command::new("docker")
        .args(["inspect", container_name])
        .output()
        .await?;

    if !output.status.success() {
        return Err(AppError::Internal(format!(
            "docker inspect failed: {}",
            String::from_utf8_lossy(&output.stderr)
        )));
    }

    Ok(String::from_utf8_lossy(&output.stdout).to_string())
}
