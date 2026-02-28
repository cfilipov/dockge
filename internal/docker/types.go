package docker

// Container holds the fields needed by handlers from a running or stopped container.
type Container struct {
    ID      string
    Name    string
    Project string // com.docker.compose.project
    Service string // com.docker.compose.service
    Image   string // image reference the container was created from
    State   string // running, exited, created, paused, dead, ...
    Health  string // healthy, unhealthy, starting, or "" (no healthcheck)
}

// ContainerBroadcast is the enriched container type sent to the frontend via
// the "containers" broadcast channel. It includes all fields needed for
// cross-store joins (networks, mounts, ports, imageId).
type ContainerBroadcast struct {
    Name        string                      `json:"name"`
    ContainerID string                      `json:"containerId"`
    ServiceName string                      `json:"serviceName"`
    StackName   string                      `json:"stackName"`
    State       string                      `json:"state"`
    Health      string                      `json:"health"`
    Image       string                      `json:"image"`
    ImageID     string                      `json:"imageId"`
    Networks    map[string]ContainerNetwork `json:"networks"`
    Mounts      []ContainerMount            `json:"mounts"`
    Ports       []ContainerPort             `json:"ports"`
}

// ContainerNetwork holds network endpoint info for a container.
type ContainerNetwork struct {
    IPv4 string `json:"ipv4"`
    IPv6 string `json:"ipv6"`
    MAC  string `json:"mac"`
}

// ContainerMount holds mount info for a container.
type ContainerMount struct {
    Name string `json:"name"`
    Type string `json:"type"` // "volume", "bind", "tmpfs"
}

// ContainerPort holds port mapping info for a container.
type ContainerPort struct {
    HostPort      uint16 `json:"hostPort"`
    ContainerPort uint16 `json:"containerPort"`
    Protocol      string `json:"protocol"` // "tcp", "udp"
}

// ContainerStat holds formatted resource-usage strings matching the Node.js frontend expectations.
type ContainerStat struct {
    Name     string `json:"Name"`
    CPUPerc  string `json:"CPUPerc"`
    MemPerc  string `json:"MemPerc"`
    MemUsage string `json:"MemUsage"`
    NetIO    string `json:"NetIO"`
    BlockIO  string `json:"BlockIO"`
    PIDs     string `json:"PIDs"`
}

// DockerEvent represents a Docker resource lifecycle event.
// Type indicates the resource kind: "container", "network", "image", "volume".
type DockerEvent struct {
    Type   string // "container", "network", "image", "volume"
    Action string // start, stop, die, create, destroy, connect, disconnect, pull, tag, ...
    // Container-specific fields (empty for non-container events)
    Project     string // from com.docker.compose.project label
    Service     string // from com.docker.compose.service label
    ContainerID string
}

// ContainerEvent represents a Docker container lifecycle event.
// Deprecated: use DockerEvent instead for new code.
type ContainerEvent = DockerEvent

// NetworkSummary holds basic info for network list display.
type NetworkSummary struct {
    Name       string            `json:"name"`
    ID         string            `json:"id"`
    Driver     string            `json:"driver"`
    Scope      string            `json:"scope"`
    Internal   bool              `json:"internal"`
    Attachable bool              `json:"attachable"`
    Ingress    bool              `json:"ingress"`
    Labels     map[string]string `json:"labels"`
}

// NetworkDetail holds full info for a single network.
type NetworkDetail struct {
    Name       string                   `json:"name"`
    ID         string                   `json:"id"`
    Driver     string                   `json:"driver"`
    Scope      string                   `json:"scope"`
    Internal   bool                     `json:"internal"`
    Attachable bool                     `json:"attachable"`
    Ingress    bool                     `json:"ingress"`
    IPv6       bool                     `json:"ipv6"`
    Created    string                   `json:"created"`
    IPAM       []NetworkIPAM            `json:"ipam"`
    Containers []NetworkContainerDetail `json:"containers"`
}

// NetworkIPAM holds IPAM configuration for a network.
type NetworkIPAM struct {
    Subnet  string `json:"subnet"`
    Gateway string `json:"gateway"`
}

// NetworkContainerDetail holds info about a container connected to a network.
type NetworkContainerDetail struct {
    Name        string `json:"name"`
    ContainerID string `json:"containerId"`
    IPv4        string `json:"ipv4"`
    IPv6        string `json:"ipv6"`
    MAC         string `json:"mac"`
    State       string `json:"state"`
}

// ImageSummary holds basic info for image list display.
type ImageSummary struct {
    ID       string   `json:"id"`
    RepoTags []string `json:"repoTags"`
    Size     string   `json:"size"`
    Created  string   `json:"created"`
    Dangling bool     `json:"dangling"`
}

// ImageDetail holds full info for a single image.
type ImageDetail struct {
    ID           string       `json:"id"`
    RepoTags     []string     `json:"repoTags"`
    Size         string       `json:"size"`
    Created      string       `json:"created"`
    Architecture string       `json:"architecture"`
    OS           string       `json:"os"`
    WorkingDir   string       `json:"workingDir"`
    Layers       []ImageLayer `json:"layers"`
}

// ImageLayer holds info about a single layer in an image's history.
type ImageLayer struct {
    ID      string `json:"id"`
    Created string `json:"created"`
    Size    string `json:"size"`
    Command string `json:"command"`
}

// ImageContainer holds info about a container using a specific image.
type ImageContainer struct {
    Name        string `json:"name"`
    ContainerID string `json:"containerId"`
    State       string `json:"state"`
}

// VolumeSummary holds basic info for volume list display.
type VolumeSummary struct {
    Name       string            `json:"name"`
    Driver     string            `json:"driver"`
    Mountpoint string            `json:"mountpoint"`
    Labels     map[string]string `json:"labels"`
}

// VolumeDetail holds full info for a single volume.
type VolumeDetail struct {
    Name       string `json:"name"`
    Driver     string `json:"driver"`
    Mountpoint string `json:"mountpoint"`
    Scope      string `json:"scope"`
    Created    string `json:"created"`
}
