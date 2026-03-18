use serde::Serialize;
use std::collections::HashMap;

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
    pub networks: HashMap<String, ContainerNetwork>,
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
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ImageSummary {
    pub id: String,
    pub repo_tags: Vec<String>,
    pub size: i64,
    pub created: i64,
}

#[derive(Debug, Clone, Serialize)]
pub struct VolumeSummary {
    pub name: String,
    pub driver: String,
    pub mountpoint: String,
}
