use crate::models::settings;
use crate::models::stack::Stack;
use crate::state::{AppState, StackUpdateInfo};
use serde_json::Value;
use sqlx::SqlitePool;
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::Semaphore;
use tracing::{debug, error, info};

const DEFAULT_CHECK_INTERVAL_HOURS: u64 = 6;
const INITIAL_DELAY_SECS: u64 = 5 * 60; // 5 minutes
const REGISTRY_TIMEOUT_SECS: u64 = 15;
const CONCURRENCY_LIMIT: usize = 3;

#[derive(Debug, Clone)]
pub struct ParsedImageRef {
    pub registry: String,
    pub repository: String,
    pub tag: String,
}

/// Parse a Docker image reference into its components
pub fn parse_image_reference(image_ref: &str) -> ParsedImageRef {
    let mut reference = image_ref.trim().to_string();

    // Handle @sha256: digest pinned images
    if let Some(pos) = reference.find("@sha256:") {
        reference = reference[..pos].to_string();
    }

    let mut registry = "registry-1.docker.io".to_string();
    let mut tag = "latest".to_string();

    // Split tag from repository
    let last_colon = reference.rfind(':');
    let last_slash = reference.rfind('/');

    if let Some(colon_pos) = last_colon {
        let slash_pos = last_slash.unwrap_or(0);
        if colon_pos > slash_pos {
            tag = reference[colon_pos + 1..].to_string();
            reference = reference[..colon_pos].to_string();
        }
    }

    // Determine if first component is a registry
    let parts: Vec<&str> = reference.split('/').collect();
    let repository = if parts.len() >= 2 && (parts[0].contains('.') || parts[0].contains(':')) {
        registry = parts[0].to_string();
        parts[1..].join("/")
    } else if parts.len() == 1 {
        format!("library/{}", parts[0])
    } else {
        reference
    };

    ParsedImageRef {
        registry,
        repository,
        tag,
    }
}

/// Fetch the remote digest for an image from a container registry
async fn fetch_remote_digest(parsed: &ParsedImageRef) -> Option<String> {
    let registry_url = if parsed.registry == "registry-1.docker.io" {
        "https://registry-1.docker.io".to_string()
    } else {
        format!("https://{}", parsed.registry)
    };

    let manifest_url = format!("{}/v2/{}/manifests/{}", registry_url, parsed.repository, parsed.tag);
    let accept = "application/vnd.docker.distribution.manifest.v2+json, \
                   application/vnd.docker.distribution.manifest.list.v2+json, \
                   application/vnd.oci.image.manifest.v1+json, \
                   application/vnd.oci.image.index.v1+json";

    let client = reqwest::Client::builder()
        .timeout(std::time::Duration::from_secs(REGISTRY_TIMEOUT_SECS))
        .build()
        .ok()?;

    // First attempt
    let res = client
        .head(&manifest_url)
        .header("Accept", accept)
        .send()
        .await
        .ok()?;

    if res.status().as_u16() == 401 {
        // Handle auth challenge
        let www_auth = res
            .headers()
            .get("www-authenticate")
            .and_then(|v| v.to_str().ok())
            .unwrap_or("")
            .to_string();

        let token = fetch_bearer_token(&www_auth, &parsed.repository).await?;

        let res = client
            .head(&manifest_url)
            .header("Accept", accept)
            .header("Authorization", format!("Bearer {}", token))
            .send()
            .await
            .ok()?;

        if !res.status().is_success() {
            return None;
        }

        return res
            .headers()
            .get("docker-content-digest")
            .and_then(|v| v.to_str().ok())
            .map(|s| s.to_string());
    }

    if !res.status().is_success() {
        return None;
    }

    res.headers()
        .get("docker-content-digest")
        .and_then(|v| v.to_str().ok())
        .map(|s| s.to_string())
}

/// Parse a Www-Authenticate header and fetch a bearer token
async fn fetch_bearer_token(www_auth: &str, repository: &str) -> Option<String> {
    let realm = regex::Regex::new(r#"realm="([^"]+)""#)
        .ok()?
        .captures(www_auth)?
        .get(1)?
        .as_str();

    let service = regex::Regex::new(r#"service="([^"]+)""#)
        .ok()
        .and_then(|re| re.captures(www_auth))
        .and_then(|c| c.get(1))
        .map(|m| m.as_str())
        .unwrap_or("");

    let scope = format!("repository:{}:pull", repository);
    let token_url = format!(
        "{}?service={}&scope={}",
        realm,
        urlencoding::encode(service),
        urlencoding::encode(&scope)
    );

    let client = reqwest::Client::builder()
        .timeout(std::time::Duration::from_secs(REGISTRY_TIMEOUT_SECS))
        .build()
        .ok()?;

    let res = client.get(&token_url).send().await.ok()?;
    if !res.status().is_success() {
        return None;
    }

    let data: Value = res.json().await.ok()?;
    data.get("token")
        .or_else(|| data.get("access_token"))
        .and_then(|v| v.as_str())
        .map(|s| s.to_string())
}

/// Get the local digest for an image via `docker image inspect`
async fn fetch_local_digest(image_ref: &str) -> Option<String> {
    let output = tokio::process::Command::new("docker")
        .args(["image", "inspect", "--format", "json", image_ref])
        .output()
        .await
        .ok()?;

    if !output.status.success() || output.stdout.is_empty() {
        return None;
    }

    let data: Value = serde_json::from_slice(&output.stdout).ok()?;
    let arr = if data.is_array() {
        data.as_array()?.clone()
    } else {
        vec![data]
    };

    if arr.is_empty() {
        return None;
    }

    let repo_digests = arr[0].get("RepoDigests")?.as_array()?;
    if repo_digests.is_empty() {
        return None;
    }

    let digest_str = repo_digests[0].as_str()?;
    let parts: Vec<&str> = digest_str.split('@').collect();
    if parts.len() >= 2 {
        Some(parts[1].to_string())
    } else {
        None
    }
}

/// Load cached results from the database into the in-memory map
pub async fn load_cache_from_db(pool: &SqlitePool) -> HashMap<String, StackUpdateInfo> {
    let mut cache = HashMap::new();

    let rows: Vec<(String, String, bool)> = sqlx::query_as(
        "SELECT stack_name, service_name, has_update FROM image_update_cache"
    )
    .fetch_all(pool)
    .await
    .unwrap_or_default();

    for (stack_name, service_name, has_update) in rows {
        let entry = cache.entry(stack_name).or_insert_with(|| StackUpdateInfo {
            has_updates: false,
            services: HashMap::new(),
        });
        entry.services.insert(service_name, has_update);
        if has_update {
            entry.has_updates = true;
        }
    }

    info!("Loaded {} cached image update entries", cache.len());
    cache
}

/// Check a single image: fetch remote + local digests, compare, write to DB
async fn check_single_image(
    pool: &SqlitePool,
    stack_name: &str,
    service_name: &str,
    image_ref: &str,
    ignore_digest: Option<&str>,
) {
    let parsed = parse_image_reference(image_ref);
    let (remote_digest, local_digest) = tokio::join!(
        fetch_remote_digest(&parsed),
        fetch_local_digest(image_ref),
    );

    let mut has_update = false;
    if let (Some(ref remote), Some(ref local)) = (&remote_digest, &local_digest) {
        has_update = remote != local;
        if has_update {
            if let Some(ignore) = ignore_digest {
                if remote == ignore {
                    has_update = false;
                }
            }
        }
    }

    let now = chrono::Utc::now().timestamp();

    // Upsert into DB
    let existing: Option<(i64,)> = sqlx::query_as(
        "SELECT id FROM image_update_cache WHERE stack_name = ? AND service_name = ?"
    )
    .bind(stack_name)
    .bind(service_name)
    .fetch_optional(pool)
    .await
    .unwrap_or(None);

    if let Some((id,)) = existing {
        let _ = sqlx::query(
            "UPDATE image_update_cache SET image_reference = ?, local_digest = ?, remote_digest = ?, has_update = ?, last_checked = ? WHERE id = ?"
        )
        .bind(image_ref)
        .bind(local_digest.as_deref())
        .bind(remote_digest.as_deref())
        .bind(has_update)
        .bind(now)
        .bind(id)
        .execute(pool)
        .await;
    } else {
        let _ = sqlx::query(
            "INSERT INTO image_update_cache (stack_name, service_name, image_reference, local_digest, remote_digest, has_update, last_checked) VALUES (?, ?, ?, ?, ?, ?, ?)"
        )
        .bind(stack_name)
        .bind(service_name)
        .bind(image_ref)
        .bind(local_digest.as_deref())
        .bind(remote_digest.as_deref())
        .bind(has_update)
        .bind(now)
        .execute(pool)
        .await;
    }

    debug!("{}/{} ({}): update={}", stack_name, service_name, image_ref, has_update);
}

/// Run a full check cycle for all stacks
pub async fn check_all(state: &Arc<AppState>) {
    // Check if enabled
    let enabled = settings::get(&state.db, "imageUpdateCheckEnabled")
        .await
        .ok()
        .flatten();
    if enabled == Some(Value::Bool(false)) {
        debug!("Image update check is disabled");
        return;
    }

    info!("Starting image update check for all stacks");

    let stack_list = match Stack::get_stack_list(&state.stacks_dir).await {
        Ok(list) => list,
        Err(e) => {
            error!("Failed to get stack list: {}", e);
            return;
        }
    };

    let semaphore = Arc::new(Semaphore::new(CONCURRENCY_LIMIT));
    let mut handles = vec![];

    for (stack_name, stack) in &stack_list {
        if !stack.is_managed_by_dockge() || stack.compose_yaml.is_empty() {
            continue;
        }

        let doc: Value = match serde_yaml::from_str(&stack.compose_yaml) {
            Ok(v) => v,
            Err(_) => continue,
        };

        let services = match doc.get("services").and_then(|s| s.as_object()) {
            Some(s) => s,
            None => continue,
        };

        for (service_name, service_config) in services {
            let image = match service_config.get("image").and_then(|i| i.as_str()) {
                Some(i) => i.to_string(),
                None => continue,
            };

            // Check dockge labels
            let labels = service_config.get("labels");
            let check_label = labels
                .and_then(|l| l.get("dockge.imageupdates.check"))
                .and_then(|v| v.as_str());
            if check_label == Some("false") {
                continue;
            }

            let ignore_digest = labels
                .and_then(|l| l.get("dockge.imageupdates.ignore"))
                .and_then(|v| v.as_str())
                .map(|s| s.to_string());

            let pool = state.db.clone();
            let stack_name = stack_name.clone();
            let service_name = service_name.clone();
            let sem = semaphore.clone();

            handles.push(tokio::spawn(async move {
                let _permit = sem.acquire().await.unwrap();
                check_single_image(
                    &pool,
                    &stack_name,
                    &service_name,
                    &image,
                    ignore_digest.as_deref(),
                )
                .await;
            }));
        }
    }

    let count = handles.len();
    for handle in handles {
        let _ = handle.await;
    }

    // Reload cache from DB
    let new_cache = load_cache_from_db(&state.db).await;
    {
        let mut cache = state.update_cache.write().await;
        *cache = new_cache;
    }

    info!("Check complete: {} images checked", count);
}

/// Check a single stack
pub async fn check_stack(state: &Arc<AppState>, stack_name: &str) {
    info!("Checking stack: {}", stack_name);

    let stack = match Stack::get_stack(&state.stacks_dir, stack_name).await {
        Ok(s) => s,
        Err(_) => return,
    };

    if !stack.is_managed_by_dockge() || stack.compose_yaml.is_empty() {
        return;
    }

    let doc: Value = match serde_yaml::from_str(&stack.compose_yaml) {
        Ok(v) => v,
        Err(_) => return,
    };

    let services = match doc.get("services").and_then(|s| s.as_object()) {
        Some(s) => s,
        None => return,
    };

    let semaphore = Arc::new(Semaphore::new(CONCURRENCY_LIMIT));
    let mut handles = vec![];

    for (service_name, service_config) in services {
        let image = match service_config.get("image").and_then(|i| i.as_str()) {
            Some(i) => i.to_string(),
            None => continue,
        };

        let labels = service_config.get("labels");
        let check_label = labels
            .and_then(|l| l.get("dockge.imageupdates.check"))
            .and_then(|v| v.as_str());
        if check_label == Some("false") {
            continue;
        }

        let ignore_digest = labels
            .and_then(|l| l.get("dockge.imageupdates.ignore"))
            .and_then(|v| v.as_str())
            .map(|s| s.to_string());

        let pool = state.db.clone();
        let stack_name = stack_name.to_string();
        let service_name = service_name.clone();
        let sem = semaphore.clone();

        handles.push(tokio::spawn(async move {
            let _permit = sem.acquire().await.unwrap();
            check_single_image(
                &pool,
                &stack_name,
                &service_name,
                &image,
                ignore_digest.as_deref(),
            )
            .await;
        }));
    }

    for handle in handles {
        let _ = handle.await;
    }

    // Reload cache from DB
    let new_cache = load_cache_from_db(&state.db).await;
    {
        let mut cache = state.update_cache.write().await;
        *cache = new_cache;
    }
}

/// Start the background update checker timer
pub fn start_background_checker(state: Arc<AppState>) {
    tokio::spawn(async move {
        // Load cache from DB immediately
        let initial_cache = load_cache_from_db(&state.db).await;
        {
            let mut cache = state.update_cache.write().await;
            *cache = initial_cache;
        }

        // Wait before first check
        tokio::time::sleep(std::time::Duration::from_secs(INITIAL_DELAY_SECS)).await;

        loop {
            check_all(&state).await;

            // Broadcast updated stack list after check
            // (handled by the caller via state.io)

            let interval_hours = settings::get(&state.db, "imageUpdateCheckInterval")
                .await
                .ok()
                .flatten()
                .and_then(|v| v.as_u64())
                .unwrap_or(DEFAULT_CHECK_INTERVAL_HOURS)
                .max(1);

            tokio::time::sleep(std::time::Duration::from_secs(interval_hours * 3600)).await;
        }
    });
}
