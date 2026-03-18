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

    pub async fn start_container(
        &self,
        name: &str,
        opts: Option<bollard::query_parameters::StartContainerOptions>,
    ) -> Result<(), bollard::errors::Error> {
        with_timeout(self.inner.start_container(name, opts)).await
    }

    pub async fn stop_container(
        &self,
        name: &str,
        opts: Option<bollard::query_parameters::StopContainerOptions>,
    ) -> Result<(), bollard::errors::Error> {
        with_timeout(self.inner.stop_container(name, opts)).await
    }

    pub async fn restart_container(
        &self,
        name: &str,
        opts: Option<bollard::query_parameters::RestartContainerOptions>,
    ) -> Result<(), bollard::errors::Error> {
        with_timeout(self.inner.restart_container(name, opts)).await
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

    ContainerBroadcast {
        id: c.id.unwrap_or_default(),
        name,
        image: c.image.unwrap_or_default(),
        state,
        status: c.status.unwrap_or_default(),
        stack_name,
        service_name,
        labels,
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

/// List all networks.
pub async fn network_list(
    docker: &DockerClient,
) -> Result<Vec<NetworkSummary>, bollard::errors::Error> {
    let networks = docker
        .list_networks(None::<bollard::query_parameters::ListNetworksOptions>)
        .await?;
    Ok(networks
        .into_iter()
        .map(|n| NetworkSummary {
            id: n.id.unwrap_or_default(),
            name: n.name.unwrap_or_default(),
            driver: n.driver.unwrap_or_default(),
            scope: n.scope.unwrap_or_default(),
        })
        .collect())
}

/// List all images.
pub async fn image_list(
    docker: &DockerClient,
) -> Result<Vec<ImageSummary>, bollard::errors::Error> {
    let images = docker
        .list_images(None::<bollard::query_parameters::ListImagesOptions>)
        .await?;
    Ok(images
        .into_iter()
        .map(|i| ImageSummary {
            id: i.id,
            repo_tags: i.repo_tags,
            size: i.size,
            created: i.created,
        })
        .collect())
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
        .map(|v| VolumeSummary {
            name: v.name,
            driver: v.driver,
            mountpoint: v.mountpoint,
        })
        .collect())
}
