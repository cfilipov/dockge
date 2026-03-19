use serde::Serialize;
use std::collections::BTreeMap;

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ContainerBroadcast {
    pub name: String,
    pub container_id: String,
    pub service_name: String,
    pub stack_name: String,
    pub state: String,
    pub health: String,
    pub image: String,
    pub image_id: String,
    pub networks: BTreeMap<String, ContainerNetwork>,
    pub mounts: Vec<ContainerMount>,
    pub ports: Vec<ContainerPort>,
}

#[derive(Debug, Clone, Serialize)]
pub struct ContainerNetwork {
    pub ipv4: String,
    pub ipv6: String,
    pub mac: String,
}

#[derive(Debug, Clone, Serialize)]
pub struct ContainerMount {
    pub name: String,
    #[serde(rename = "type")]
    pub mount_type: String,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ContainerPort {
    pub host_port: u16,
    pub container_port: u16,
    pub protocol: String,
}

#[derive(Debug, Clone, Serialize)]
pub struct NetworkSummary {
    pub id: String,
    pub name: String,
    pub driver: String,
    pub scope: String,
    pub internal: bool,
    pub attachable: bool,
    pub ingress: bool,
    pub labels: BTreeMap<String, String>,
}

#[derive(Debug, Clone, Serialize)]
pub struct NetworkDetail {
    #[serde(flatten)]
    pub summary: NetworkSummary,
    pub ipv6: bool,
    pub created: String,
    pub ipam: Vec<NetworkIPAM>,
    pub containers: Vec<NetworkContainerDetail>,
}

#[derive(Debug, Clone, Serialize)]
pub struct NetworkIPAM {
    pub subnet: String,
    pub gateway: String,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct NetworkContainerDetail {
    pub name: String,
    pub container_id: String,
    pub ipv4: String,
    pub ipv6: String,
    pub mac: String,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ImageSummary {
    pub id: String,
    pub repo_tags: Vec<String>,
    pub size: String,
    pub created: String,
    pub dangling: bool,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ImageDetail {
    #[serde(flatten)]
    pub summary: ImageSummary,
    pub architecture: String,
    pub os: String,
    pub working_dir: String,
    pub layers: Vec<ImageLayer>,
}

#[derive(Debug, Clone, Serialize)]
pub struct ImageLayer {
    pub id: String,
    pub created: String,
    pub size: String,
    pub command: String,
}

#[derive(Debug, Clone, Serialize)]
pub struct VolumeSummary {
    pub name: String,
    pub driver: String,
    pub mountpoint: String,
    pub labels: BTreeMap<String, String>,
}

#[derive(Debug, Clone, Serialize)]
pub struct VolumeDetail {
    #[serde(flatten)]
    pub summary: VolumeSummary,
    pub scope: String,
    pub created: String,
}
