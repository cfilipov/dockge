pub mod types;

use bollard::container::LogOutput;
use bollard::Docker;
use futures_util::Stream;
use std::collections::HashMap;
use std::time::Duration;
use types::*;

const DOCKER_TIMEOUT: Duration = Duration::from_secs(10);

/// Docker client with automatic timeouts on all one-shot operations.
///
/// Streaming calls (`logs`, `events`, `stats`) pass through without timeout.
/// Callers cannot bypass the timeout by accident — the inner client is private.
#[derive(Clone)]
pub struct DockerClient {
    inner: Docker,
}

impl DockerClient {
    pub fn new(inner: Docker) -> Self {
        Self { inner }
    }

    /// Escape hatch for APIs not yet exposed as methods.
    #[allow(dead_code)]
    pub(crate) fn raw(&self) -> &Docker {
        &self.inner
    }

    // ── One-shot operations (all wrapped with timeout) ──────────────────

    pub async fn list_containers(
        &self,
        opts: Option<bollard::query_parameters::ListContainersOptions>,
    ) -> Result<Vec<bollard::models::ContainerSummary>, bollard::errors::Error> {
        with_timeout(self.inner.list_containers(opts)).await
    }

    pub async fn inspect_container(
        &self,
        name: &str,
        opts: Option<bollard::query_parameters::InspectContainerOptions>,
    ) -> Result<bollard::models::ContainerInspectResponse, bollard::errors::Error> {
        with_timeout(self.inner.inspect_container(name, opts)).await
    }

    pub async fn inspect_network(
        &self,
        name: &str,
        opts: Option<bollard::query_parameters::InspectNetworkOptions>,
    ) -> Result<bollard::models::Network, bollard::errors::Error> {
        with_timeout(self.inner.inspect_network(name, opts)).await
    }

    pub async fn inspect_image(
        &self,
        name: &str,
    ) -> Result<bollard::models::ImageInspect, bollard::errors::Error> {
        with_timeout(self.inner.inspect_image(name)).await
    }

    pub async fn image_history(
        &self,
        name: &str,
    ) -> Result<Vec<bollard::models::HistoryResponseItem>, bollard::errors::Error> {
        with_timeout(self.inner.image_history(name)).await
    }

    pub async fn inspect_volume(
        &self,
        name: &str,
    ) -> Result<bollard::models::Volume, bollard::errors::Error> {
        with_timeout(self.inner.inspect_volume(name)).await
    }

    pub async fn list_networks(
        &self,
        opts: Option<bollard::query_parameters::ListNetworksOptions>,
    ) -> Result<Vec<bollard::models::Network>, bollard::errors::Error> {
        with_timeout(self.inner.list_networks(opts)).await
    }

    pub async fn list_images(
        &self,
        opts: Option<bollard::query_parameters::ListImagesOptions>,
    ) -> Result<Vec<bollard::models::ImageSummary>, bollard::errors::Error> {
        with_timeout(self.inner.list_images(opts)).await
    }

    pub async fn list_volumes(
        &self,
        opts: Option<bollard::query_parameters::ListVolumesOptions>,
    ) -> Result<bollard::models::VolumeListResponse, bollard::errors::Error> {
        with_timeout(self.inner.list_volumes(opts)).await
    }

    pub async fn inspect_registry_image(
        &self,
        image: &str,
        credentials: Option<bollard::auth::DockerCredentials>,
    ) -> Result<bollard::models::DistributionInspect, bollard::errors::Error> {
        with_timeout(self.inner.inspect_registry_image(image, credentials)).await
    }

    pub async fn top_processes(
        &self,
        container: &str,
        opts: Option<bollard::query_parameters::TopOptions>,
    ) -> Result<bollard::models::ContainerTopResponse, bollard::errors::Error> {
        with_timeout(self.inner.top_processes(container, opts)).await
    }

    // ── Streaming operations (no timeout) ───────────────────────────────

    pub fn logs(
        &self,
        container_name: &str,
        opts: Option<bollard::query_parameters::LogsOptions>,
    ) -> impl Stream<Item = Result<LogOutput, bollard::errors::Error>> {
        self.inner.logs(container_name, opts)
    }

    pub fn events(
        &self,
        opts: Option<bollard::query_parameters::EventsOptions>,
    ) -> impl Stream<Item = Result<bollard::models::EventMessage, bollard::errors::Error>> {
        self.inner.events(opts)
    }

    pub fn stats(
        &self,
        container: &str,
        opts: Option<bollard::query_parameters::StatsOptions>,
    ) -> impl Stream<Item = Result<bollard::models::ContainerStatsResponse, bollard::errors::Error>>
    {
        self.inner.stats(container, opts)
    }
}

/// Apply a 10s timeout to a Docker API future.
async fn with_timeout<T>(
    fut: impl std::future::Future<Output = Result<T, bollard::errors::Error>>,
) -> Result<T, bollard::errors::Error> {
    match tokio::time::timeout(DOCKER_TIMEOUT, fut).await {
        Ok(result) => result,
        Err(_) => Err(bollard::errors::Error::RequestTimeoutError),
    }
}

/// Create a Docker client from DOCKER_HOST or the default socket.
/// Uses API version 1.47 to match the mock daemon's target version.
pub fn connect() -> Result<DockerClient, bollard::errors::Error> {
    let host = std::env::var("DOCKER_HOST")
        .unwrap_or_else(|_| "unix:///var/run/docker.sock".to_string());
    let inner = if host.starts_with("unix://") {
        Docker::connect_with_unix(
            &host,
            120,
            &bollard::ClientVersion {
                major_version: 1,
                minor_version: 47,
            },
        )?
    } else {
        Docker::connect_with_defaults()?
    };
    Ok(DockerClient::new(inner))
}

/// List all containers (optionally filtered by compose project/stack name).
pub async fn container_list(
    docker: &DockerClient,
    stack_filter: Option<&str>,
) -> Result<Vec<ContainerBroadcast>, bollard::errors::Error> {
    let opts = if let Some(stack) = stack_filter {
        let mut filters = HashMap::new();
        filters.insert(
            "label".to_string(),
            vec![format!("com.docker.compose.project={stack}")],
        );
        bollard::query_parameters::ListContainersOptionsBuilder::default()
            .all(true)
            .filters(&filters)
            .build()
    } else {
        bollard::query_parameters::ListContainersOptionsBuilder::default()
            .all(true)
            .build()
    };

    let containers = docker.list_containers(Some(opts)).await?;

    Ok(containers.into_iter().map(container_from_bollard).collect())
}

/// Parse health status from Docker's human-readable Status string.
/// Returns "healthy", "unhealthy", "starting", or "" if no healthcheck.
fn parse_health_from_status(state: &str, status: &str) -> String {
    if state != "running" || status.is_empty() {
        return String::new();
    }
    let lower = status.to_lowercase();
    if lower.ends_with("(unhealthy)") {
        "unhealthy".to_string()
    } else if lower.ends_with("(healthy)") {
        "healthy".to_string()
    } else if lower.ends_with("(health: starting)") {
        "starting".to_string()
    } else {
        String::new()
    }
}

/// Map a bollard ContainerSummary to our ContainerBroadcast type.
fn container_from_bollard(c: bollard::models::ContainerSummary) -> ContainerBroadcast {
    let labels = c.labels.unwrap_or_default();
    let stack_name = labels
        .get("com.docker.compose.project")
        .cloned()
        .unwrap_or_default();
    let service_name = labels
        .get("com.docker.compose.service")
        .cloned()
        .unwrap_or_default();

    let name = c
        .names
        .and_then(|v| v.into_iter().next())
        .map(|n| match n.strip_prefix('/') {
            Some(s) => s.to_string(),
            None => n,
        })
        .unwrap_or_default();

    let state = c
        .state
        .as_ref()
        .map(|s| s.as_ref().to_string())
        .unwrap_or_default();

    let status = c.status.as_deref().unwrap_or_default();
    let health = parse_health_from_status(&state, status);

    // Extract network endpoints
    let networks = c
        .network_settings
        .and_then(|ns| ns.networks)
        .unwrap_or_default()
        .into_iter()
        .map(|(net_name, ep)| {
            (
                net_name,
                ContainerNetwork {
                    ipv4: ep.ip_address.unwrap_or_default(),
                    ipv6: ep.global_ipv6_address.unwrap_or_default(),
                    mac: ep.mac_address.unwrap_or_default(),
                },
            )
        })
        .collect();

    // Extract mounts
    let mounts = c
        .mounts
        .unwrap_or_default()
        .into_iter()
        .map(|m| ContainerMount {
            name: m.name.unwrap_or_default(),
            mount_type: m.typ.map(|t| t.to_string()).unwrap_or_default(),
        })
        .collect();

    // Extract ports
    let ports = c
        .ports
        .unwrap_or_default()
        .into_iter()
        .map(|p| ContainerPort {
            host_port: p.public_port.unwrap_or(0),
            container_port: p.private_port,
            protocol: p.typ.map(|t| t.to_string()).unwrap_or_default(),
        })
        .collect();

    ContainerBroadcast {
        name,
        container_id: c.id.unwrap_or_default(),
        service_name,
        stack_name,
        state,
        health,
        image: c.image.unwrap_or_default(),
        image_id: c.image_id.unwrap_or_default(),
        networks,
        mounts,
        ports,
    }
}

/// List containers filtered by a set of container IDs.
pub async fn container_list_by_ids(
    docker: &DockerClient,
    ids: &std::collections::HashSet<String>,
) -> Result<Vec<ContainerBroadcast>, bollard::errors::Error> {
    if ids.is_empty() {
        return Ok(Vec::new());
    }

    let mut filters = HashMap::new();
    filters.insert(
        "id".to_string(),
        ids.iter().cloned().collect::<Vec<_>>(),
    );
    let opts = bollard::query_parameters::ListContainersOptionsBuilder::default()
        .all(true)
        .filters(&filters)
        .build();

    let containers = docker.list_containers(Some(opts)).await?;

    Ok(containers.into_iter().map(container_from_bollard).collect())
}

// ── Network helpers ─────────────────────────────────────────────────────────

/// Map a bollard Network to our NetworkSummary type.
fn network_from_bollard(n: bollard::models::Network) -> NetworkSummary {
    NetworkSummary {
        id: n.id.unwrap_or_default(),
        name: n.name.unwrap_or_default(),
        driver: n.driver.unwrap_or_default(),
        scope: n.scope.unwrap_or_default(),
        internal: n.internal.unwrap_or_default(),
        attachable: n.attachable.unwrap_or_default(),
        ingress: n.ingress.unwrap_or_default(),
        labels: n.labels.unwrap_or_default().into_iter().collect(),
    }
}

/// List all networks.
pub async fn network_list(
    docker: &DockerClient,
) -> Result<Vec<NetworkSummary>, bollard::errors::Error> {
    let networks = docker
        .list_networks(None::<bollard::query_parameters::ListNetworksOptions>)
        .await?;
    Ok(networks.into_iter().map(network_from_bollard).collect())
}

/// List networks filtered by IDs. Returns empty vec if no IDs given.
pub async fn network_list_by_ids(
    docker: &DockerClient,
    ids: &std::collections::HashSet<String>,
) -> Result<Vec<NetworkSummary>, bollard::errors::Error> {
    if ids.is_empty() {
        return Ok(Vec::new());
    }

    let mut filters = HashMap::new();
    filters.insert(
        "id".to_string(),
        ids.iter().cloned().collect::<Vec<_>>(),
    );
    let opts = bollard::query_parameters::ListNetworksOptionsBuilder::default()
        .filters(&filters)
        .build();

    let networks = docker.list_networks(Some(opts)).await?;
    Ok(networks.into_iter().map(network_from_bollard).collect())
}

/// Inspect a single network and return a shaped response matching the Go backend.
pub async fn network_inspect(
    docker: &DockerClient,
    name: &str,
) -> Result<NetworkDetail, bollard::errors::Error> {
    let raw = docker
        .inspect_network(name, None::<bollard::query_parameters::InspectNetworkOptions>)
        .await?;

    // Flatten IPAM configs into Vec<NetworkIPAM>
    let ipam = raw
        .ipam
        .as_ref()
        .and_then(|ipam| ipam.config.as_ref())
        .map(|configs| {
            configs
                .iter()
                .map(|cfg| NetworkIPAM {
                    subnet: cfg.subnet.clone().unwrap_or_default(),
                    gateway: cfg.gateway.clone().unwrap_or_default(),
                })
                .collect()
        })
        .unwrap_or_default();

    // Extract containers map into sorted Vec<NetworkContainerDetail>
    let mut containers: Vec<NetworkContainerDetail> = raw
        .containers
        .as_ref()
        .map(|map| {
            map.iter()
                .map(|(id, ep)| NetworkContainerDetail {
                    name: ep.name.clone().unwrap_or_default(),
                    container_id: id.clone(),
                    ipv4: ep.ipv4_address.clone().unwrap_or_default(),
                    ipv6: ep.ipv6_address.clone().unwrap_or_default(),
                    mac: ep.mac_address.clone().unwrap_or_default(),
                })
                .collect()
        })
        .unwrap_or_default();
    containers.sort_by(|a, b| a.name.cmp(&b.name));

    // Format created time — bollard gives us a String already
    let created = raw.created.unwrap_or_default();

    Ok(NetworkDetail {
        summary: NetworkSummary {
            id: raw.id.unwrap_or_default(),
            name: raw.name.unwrap_or_default(),
            driver: raw.driver.unwrap_or_default(),
            scope: raw.scope.unwrap_or_default(),
            internal: raw.internal.unwrap_or_default(),
            attachable: raw.attachable.unwrap_or_default(),
            ingress: raw.ingress.unwrap_or_default(),
            labels: raw.labels.unwrap_or_default().into_iter().collect(),
        },
        ipv6: raw.enable_ipv6.unwrap_or_default(),
        created,
        ipam,
        containers,
    })
}

// ── Image helpers ───────────────────────────────────────────────────────────

/// Map a bollard ImageSummary to our ImageSummary type.
fn image_from_bollard(i: bollard::models::ImageSummary) -> ImageSummary {
    let tags: Vec<String> = i
        .repo_tags
        .into_iter()
        .filter(|t| t != "<none>:<none>")
        .collect();
    let dangling = tags.is_empty();
    ImageSummary {
        id: i.id,
        repo_tags: tags,
        size: format_bytes(i.size as u64),
        created: format_unix_timestamp(i.created),
        dangling,
    }
}

/// List all images.
pub async fn image_list(
    docker: &DockerClient,
) -> Result<Vec<ImageSummary>, bollard::errors::Error> {
    let images = docker
        .list_images(None::<bollard::query_parameters::ListImagesOptions>)
        .await?;
    let mut result: Vec<ImageSummary> = images.into_iter().map(image_from_bollard).collect();
    result.sort_by(|a, b| a.id.cmp(&b.id));
    Ok(result)
}

/// List images filtered by IDs (uses `reference` filter, matching Go backend).
/// Returns empty vec if no IDs given.
pub async fn image_list_by_ids(
    docker: &DockerClient,
    ids: &std::collections::HashSet<String>,
) -> Result<Vec<ImageSummary>, bollard::errors::Error> {
    if ids.is_empty() {
        return Ok(Vec::new());
    }

    let mut filters = HashMap::new();
    filters.insert(
        "reference".to_string(),
        ids.iter().cloned().collect::<Vec<_>>(),
    );
    let opts = bollard::query_parameters::ListImagesOptionsBuilder::default()
        .filters(&filters)
        .build();

    let images = docker.list_images(Some(opts)).await?;
    let mut result: Vec<ImageSummary> = images.into_iter().map(image_from_bollard).collect();
    result.sort_by(|a, b| a.id.cmp(&b.id));
    Ok(result)
}

/// Inspect a single image and return a shaped response matching the Go backend.
pub async fn image_inspect_detail(
    docker: &DockerClient,
    image_ref: &str,
) -> Result<ImageDetail, bollard::errors::Error> {
    let raw = docker.inspect_image(image_ref).await?;
    let history = docker.image_history(image_ref).await?;

    let layers: Vec<ImageLayer> = history
        .into_iter()
        .map(|h| {
            let id = if h.id == "<missing>" || h.id.is_empty() {
                "<missing>".to_string()
            } else if h.id.len() > 12 {
                h.id[..12].to_string()
            } else {
                h.id
            };
            ImageLayer {
                id,
                created: format_unix_timestamp(h.created),
                size: format_bytes(h.size as u64),
                command: h.created_by,
            }
        })
        .collect();

    let tags: Vec<String> = raw
        .repo_tags
        .unwrap_or_default()
        .into_iter()
        .filter(|t| t != "<none>:<none>")
        .collect();
    let dangling = tags.is_empty();

    let working_dir = raw
        .config
        .as_ref()
        .and_then(|c| c.working_dir.clone())
        .unwrap_or_default();

    let size = raw.size.unwrap_or_default();
    let created = raw.created.unwrap_or_default();

    Ok(ImageDetail {
        summary: ImageSummary {
            id: raw.id.unwrap_or_default(),
            repo_tags: tags,
            size: format_bytes(size as u64),
            created,
            dangling,
        },
        architecture: raw.architecture.unwrap_or_default(),
        os: raw.os.unwrap_or_default(),
        working_dir,
        layers,
    })
}

/// Options for container log streaming.
#[derive(Debug, Default)]
pub struct ContainerLogsOpts {
    pub follow: bool,
    pub stdout: bool,
    pub stderr: bool,
    pub timestamps: bool,
    pub tail: String,
}

/// Stream container logs via the Docker SDK.
pub fn container_logs(
    docker: &DockerClient,
    container_name: &str,
    opts: ContainerLogsOpts,
) -> impl Stream<Item = Result<LogOutput, bollard::errors::Error>> {
    let bollard_opts = bollard::query_parameters::LogsOptionsBuilder::default()
        .follow(opts.follow)
        .stdout(opts.stdout)
        .stderr(opts.stderr)
        .timestamps(opts.timestamps)
        .tail(if opts.tail.is_empty() { "all" } else { &opts.tail })
        .build();
    docker.logs(container_name, Some(bollard_opts))
}

// ── Volume helpers ──────────────────────────────────────────────────────────

/// Map a bollard Volume to our VolumeSummary type.
fn volume_from_bollard(v: bollard::models::Volume) -> VolumeSummary {
    VolumeSummary {
        name: v.name,
        driver: v.driver,
        mountpoint: v.mountpoint,
        labels: v.labels.into_iter().collect(),
    }
}

/// List all volumes.
pub async fn volume_list(
    docker: &DockerClient,
) -> Result<Vec<VolumeSummary>, bollard::errors::Error> {
    let resp = docker
        .list_volumes(None::<bollard::query_parameters::ListVolumesOptions>)
        .await?;
    Ok(resp
        .volumes
        .unwrap_or_default()
        .into_iter()
        .map(volume_from_bollard)
        .collect())
}

/// List volumes filtered by names. Returns empty vec if no names given.
pub async fn volume_list_by_names(
    docker: &DockerClient,
    names: &std::collections::HashSet<String>,
) -> Result<Vec<VolumeSummary>, bollard::errors::Error> {
    if names.is_empty() {
        return Ok(Vec::new());
    }

    let mut filters = HashMap::new();
    filters.insert(
        "name".to_string(),
        names.iter().cloned().collect::<Vec<_>>(),
    );
    let opts = bollard::query_parameters::ListVolumesOptionsBuilder::default()
        .filters(&filters)
        .build();

    let resp = docker.list_volumes(Some(opts)).await?;
    Ok(resp
        .volumes
        .unwrap_or_default()
        .into_iter()
        .map(volume_from_bollard)
        .collect())
}

/// Inspect a single volume and return a shaped response matching the Go backend.
pub async fn volume_inspect(
    docker: &DockerClient,
    name: &str,
) -> Result<VolumeDetail, bollard::errors::Error> {
    let raw = docker.inspect_volume(name).await?;

    let scope = raw
        .scope
        .map(|s| s.to_string())
        .unwrap_or_default();
    let created = raw.created_at.unwrap_or_default();

    Ok(VolumeDetail {
        summary: VolumeSummary {
            name: raw.name,
            driver: raw.driver,
            mountpoint: raw.mountpoint,
            labels: raw.labels.into_iter().collect(),
        },
        scope,
        created,
    })
}

/// Format a byte count as a human-readable string (e.g. "1.5MiB").
pub(crate) fn format_bytes(b: u64) -> String {
    const UNIT: u64 = 1024;
    if b < UNIT {
        return format!("{b}B");
    }
    let mut div = UNIT;
    let mut exp = 0usize;
    let mut n = b / UNIT;
    while n >= UNIT {
        div *= UNIT;
        exp += 1;
        n /= UNIT;
    }
    let units = b"KMGTPE";
    format!("{:.1}{}iB", b as f64 / div as f64, units[exp] as char)
}

/// Format a Unix timestamp (seconds) as an RFC3339 string.
fn format_unix_timestamp(secs: i64) -> String {
    use std::fmt::Write;
    // Simple UTC conversion without pulling in chrono
    const SECS_PER_DAY: i64 = 86400;
    const DAYS_PER_400Y: i64 = 146097;
    const DAYS_PER_100Y: i64 = 36524;
    const DAYS_PER_4Y: i64 = 1461;

    let total_secs = secs;
    let day_secs = ((total_secs % SECS_PER_DAY) + SECS_PER_DAY) % SECS_PER_DAY;
    let hour = day_secs / 3600;
    let min = (day_secs % 3600) / 60;
    let sec = day_secs % 60;

    // Days since 1970-01-01
    let mut days = (total_secs - day_secs) / SECS_PER_DAY;
    // Shift to 2000-03-01 epoch
    days += 719468;

    let era = if days >= 0 { days } else { days - DAYS_PER_400Y + 1 } / DAYS_PER_400Y;
    let doe = days - era * DAYS_PER_400Y; // day of era [0, 146096]
    let yoe = (doe - doe / (DAYS_PER_4Y - 1) + doe / DAYS_PER_100Y - doe / (DAYS_PER_400Y - 1)) / 365;
    let y = yoe + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let d = doy - (153 * mp + 2) / 5 + 1;
    let m = if mp < 10 { mp + 3 } else { mp - 9 };
    let y = if m <= 2 { y + 1 } else { y };

    let mut buf = String::with_capacity(20);
    let _ = write!(buf, "{y:04}-{m:02}-{d:02}T{hour:02}:{min:02}:{sec:02}Z");
    buf
}

#[cfg(test)]
mod tests {
    use super::*;

    // ── format_bytes ────────────────────────────────────────────────────

    #[test]
    fn format_bytes_zero() {
        assert_eq!(format_bytes(0), "0B");
    }

    #[test]
    fn format_bytes_below_unit() {
        assert_eq!(format_bytes(512), "512B");
        assert_eq!(format_bytes(1023), "1023B");
    }

    #[test]
    fn format_bytes_exact_kib() {
        assert_eq!(format_bytes(1024), "1.0KiB");
    }

    #[test]
    fn format_bytes_fractional_kib() {
        assert_eq!(format_bytes(1536), "1.5KiB");
    }

    #[test]
    fn format_bytes_exact_mib() {
        assert_eq!(format_bytes(1_048_576), "1.0MiB");
    }

    #[test]
    fn format_bytes_exact_gib() {
        assert_eq!(format_bytes(1_073_741_824), "1.0GiB");
    }

    #[test]
    fn format_bytes_exact_tib() {
        assert_eq!(format_bytes(1_099_511_627_776), "1.0TiB");
    }

    // ── format_unix_timestamp ───────────────────────────────────────────

    #[test]
    fn format_unix_timestamp_epoch() {
        assert_eq!(format_unix_timestamp(0), "1970-01-01T00:00:00Z");
    }

    #[test]
    fn format_unix_timestamp_known_date() {
        assert_eq!(format_unix_timestamp(1704067200), "2024-01-01T00:00:00Z");
    }

    #[test]
    fn format_unix_timestamp_leap_year() {
        assert_eq!(format_unix_timestamp(1709208000), "2024-02-29T12:00:00Z");
    }

    #[test]
    fn format_unix_timestamp_end_of_day() {
        assert_eq!(format_unix_timestamp(1735689599), "2024-12-31T23:59:59Z");
    }

    #[test]
    fn format_unix_timestamp_y2k() {
        assert_eq!(format_unix_timestamp(946684800), "2000-01-01T00:00:00Z");
    }

    #[test]
    fn format_unix_timestamp_negative() {
        assert_eq!(format_unix_timestamp(-1), "1969-12-31T23:59:59Z");
    }

    #[test]
    fn format_unix_timestamp_pre_epoch() {
        assert_eq!(format_unix_timestamp(-315619200), "1960-01-01T00:00:00Z");
    }

    // ── parse_health_from_status ────────────────────────────────────────

    #[test]
    fn health_healthy() {
        assert_eq!(parse_health_from_status("running", "Up 2 hours (healthy)"), "healthy");
    }

    #[test]
    fn health_unhealthy() {
        assert_eq!(parse_health_from_status("running", "Up 5 minutes (unhealthy)"), "unhealthy");
    }

    #[test]
    fn health_starting() {
        assert_eq!(parse_health_from_status("running", "Up 10 seconds (health: starting)"), "starting");
    }

    #[test]
    fn health_no_healthcheck() {
        assert_eq!(parse_health_from_status("running", "Up 2 hours"), "");
    }

    #[test]
    fn health_not_running() {
        assert_eq!(parse_health_from_status("exited", "Exited (0) 5 minutes ago"), "");
    }

    #[test]
    fn health_empty_status() {
        assert_eq!(parse_health_from_status("running", ""), "");
    }

    #[test]
    fn health_stopped_state() {
        assert_eq!(parse_health_from_status("stopped", ""), "");
    }

    // ── container_from_bollard ──────────────────────────────────────────

    #[test]
    fn container_from_bollard_all_none() {
        let c = bollard::models::ContainerSummary {
            id: None,
            names: None,
            image: None,
            image_id: None,
            command: None,
            created: None,
            ports: None,
            size_rw: None,
            size_root_fs: None,
            labels: None,
            state: None,
            status: None,
            host_config: None,
            network_settings: None,
            mounts: None,
            image_manifest_descriptor: None,
        };
        let result = container_from_bollard(c);
        assert_eq!(result.name, "");
        assert_eq!(result.container_id, "");
        assert_eq!(result.state, "");
        assert_eq!(result.health, "");
        assert_eq!(result.image, "");
    }
}
