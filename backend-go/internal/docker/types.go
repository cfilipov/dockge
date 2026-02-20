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
