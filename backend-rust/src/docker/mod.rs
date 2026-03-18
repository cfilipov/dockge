pub mod types;

use bollard::container::LogOutput;
use bollard::Docker;
use futures_util::Stream;
use std::collections::HashMap;
use types::*;

/// Create a Docker client from DOCKER_HOST or the default socket.
/// Uses API version 1.47 to match the mock daemon's target version.
pub fn connect() -> Result<Docker, bollard::errors::Error> {
    let host = std::env::var("DOCKER_HOST")
        .unwrap_or_else(|_| "unix:///var/run/docker.sock".to_string());
    if host.starts_with("unix://") {
        Docker::connect_with_unix(
            &host,
            120,
            &bollard::ClientVersion {
                major_version: 1,
                minor_version: 47,
            },
        )
    } else {
        Docker::connect_with_defaults()
    }
}

/// List all containers (optionally filtered by compose project/stack name).
pub async fn container_list(
    docker: &Docker,
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
    let labels = c.labels.clone().unwrap_or_default();
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
        .as_ref()
        .and_then(|n| n.first())
        .map(|n| n.trim_start_matches('/').to_string())
        .unwrap_or_default();

    let state = c
        .state
        .map(|s| format!("{:?}", s).to_lowercase())
        .unwrap_or_default();

    ContainerBroadcast {
        id: c.id.clone().unwrap_or_default(),
        name,
        image: c.image.clone().unwrap_or_default(),
        state,
        status: c.status.clone().unwrap_or_default(),
        stack_name,
        service_name,
        labels,
    }
}

/// List containers filtered by a set of container IDs.
pub async fn container_list_by_ids(
    docker: &Docker,
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
    docker: &Docker,
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
    docker: &Docker,
) -> Result<Vec<ImageSummary>, bollard::errors::Error> {
    let images = docker
        .list_images(None::<bollard::query_parameters::ListImagesOptions>)
        .await?;
    Ok(images
        .into_iter()
        .map(|i| ImageSummary {
            id: i.id.clone(),
            repo_tags: i.repo_tags.clone(),
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
    docker: &Docker,
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
    docker: &Docker,
) -> Result<Vec<VolumeSummary>, bollard::errors::Error> {
    let resp = docker
        .list_volumes(None::<bollard::query_parameters::ListVolumesOptions>)
        .await?;
    Ok(resp
        .volumes
        .unwrap_or_default()
        .into_iter()
        .map(|v| VolumeSummary {
            name: v.name.clone(),
            driver: v.driver.clone(),
            mountpoint: v.mountpoint.clone(),
        })
        .collect())
}
