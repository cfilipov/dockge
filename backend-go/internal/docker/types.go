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

// ContainerEvent represents a Docker container lifecycle event.
type ContainerEvent struct {
    Action      string // start, stop, die, pause, unpause, health_status, destroy, ...
    Project     string // from com.docker.compose.project label
    Service     string // from com.docker.compose.service label
    ContainerID string
}

// NetworkSummary holds basic info for network list display.
type NetworkSummary struct {
    Name       string `json:"name"`
    ID         string `json:"id"`
    Driver     string `json:"driver"`
    Scope      string `json:"scope"`
    Internal   bool   `json:"internal"`
    Attachable bool   `json:"attachable"`
    Ingress    bool   `json:"ingress"`
    Containers int    `json:"containers"`
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
}

// ImageSummary holds basic info for image list display.
type ImageSummary struct {
    ID         string   `json:"id"`
    RepoTags   []string `json:"repoTags"`
    Size       string   `json:"size"`
    Created    string   `json:"created"`
    Containers int      `json:"containers"`
    Dangling   bool     `json:"dangling"`
}

// ImageDetail holds full info for a single image.
type ImageDetail struct {
    ID           string           `json:"id"`
    RepoTags     []string         `json:"repoTags"`
    Size         string           `json:"size"`
    Created      string           `json:"created"`
    Architecture string           `json:"architecture"`
    OS           string           `json:"os"`
    WorkingDir   string           `json:"workingDir"`
    Layers       []ImageLayer     `json:"layers"`
    Containers   []ImageContainer `json:"containers"`
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
