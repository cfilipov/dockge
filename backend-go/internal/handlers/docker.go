package handlers

import (
    "context"
    "log/slog"
    "strings"
    "time"

    "github.com/cfilipov/dockge/backend-go/internal/docker"
    "github.com/cfilipov/dockge/backend-go/internal/ws"
)

func RegisterDockerHandlers(app *App) {
    app.WS.Handle("serviceStatusList", app.handleServiceStatusList)
    app.WS.Handle("dockerStats", app.handleDockerStats)
    app.WS.Handle("containerInspect", app.handleContainerInspect)
    app.WS.Handle("containerTop", app.handleContainerTop)
    app.WS.Handle("getDockerNetworkList", app.handleGetDockerNetworkList)
    app.WS.Handle("networkInspect", app.handleNetworkInspect)
    app.WS.Handle("requestContainerList", app.handleRequestContainerList)
    app.WS.Handle("getDockerImageList", app.handleGetDockerImageList)
    app.WS.Handle("imageInspect", app.handleImageInspect)
    app.WS.Handle("getDockerVolumeList", app.handleGetDockerVolumeList)
    app.WS.Handle("volumeInspect", app.handleVolumeInspect)
}

// ServiceEntry represents a single container's status within a service.
type ServiceEntry struct {
    Status string `json:"status"`
    Name   string `json:"name"`
    Image  string `json:"image"`
}

// serviceStatusResponse is the typed response for serviceStatusList.
type serviceStatusResponse struct {
    OK                    bool                       `json:"ok"`
    ServiceStatusList     map[string][]ServiceEntry  `json:"serviceStatusList"`
    ServiceUpdateStatus   map[string]bool            `json:"serviceUpdateStatus"`
    ServiceRecreateStatus map[string]bool            `json:"serviceRecreateStatus"`
}

// handleServiceStatusList returns per-service status from the in-memory container
// cache (populated by the stack watcher). Falls back to a live ContainerList query
// only if the cache has no data for this stack.
func (app *App) handleServiceStatusList(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    args := parseArgs(msg)
    stackName := argString(args, 0)
    if stackName == "" {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack name required"})
        }
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Query containers for this stack via the Docker client
    containers, err := app.Docker.ContainerList(ctx, true, stackName)
    if err != nil {
        slog.Warn("serviceStatusList", "err", err, "stack", stackName)
        if msg.ID != nil {
            c.SendAck(*msg.ID, serviceStatusResponse{
                OK:                    true,
                ServiceStatusList:     map[string][]ServiceEntry{},
                ServiceUpdateStatus:   map[string]bool{},
                ServiceRecreateStatus: map[string]bool{},
            })
        }
        return
    }

    serviceStatusList, runningImages := containersToServiceStatus(containers)

    // Compare running images vs compose.yaml to compute recreateNecessary per service
    composeImages := app.ComposeCache.GetImages(stackName)
    serviceRecreateStatus := make(map[string]bool, len(runningImages))
    anyRecreate := false
    for svc, runningImage := range runningImages {
        composeImage, ok := composeImages[svc]
        if ok && runningImage != "" && composeImage != "" && runningImage != composeImage {
            serviceRecreateStatus[svc] = true
            anyRecreate = true
        } else {
            serviceRecreateStatus[svc] = false
        }
    }
    app.SetRecreateNecessary(stackName, anyRecreate)

    // Per-service image update status from BBolt cache
    serviceUpdateStatus := make(map[string]bool)
    if svcUpdates, err := app.ImageUpdates.ServiceUpdatesForStack(stackName); err == nil {
        serviceUpdateStatus = svcUpdates
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, serviceStatusResponse{
            OK:                    true,
            ServiceStatusList:     serviceStatusList,
            ServiceUpdateStatus:   serviceUpdateStatus,
            ServiceRecreateStatus: serviceRecreateStatus,
        })
    }
}

// containersToServiceStatus converts a list of containers (from the Docker client)
// into the serviceStatusList map and a running-images map, matching the format
// the frontend expects.
func containersToServiceStatus(containers []docker.Container) (map[string][]ServiceEntry, map[string]string) {
    result := make(map[string][]ServiceEntry, len(containers))
    runningImages := make(map[string]string, len(containers))

    for _, c := range containers {
        serviceName := c.Service
        if serviceName == "" {
            serviceName = extractServiceName(c.Name)
        }
        if serviceName == "" {
            continue
        }

        status := "unknown"
        if c.Health != "" {
            status = strings.ToLower(c.Health)
        } else if c.State != "" {
            status = strings.ToLower(c.State)
        }

        runningImages[serviceName] = c.Image

        entry := ServiceEntry{
            Status: status,
            Name:   c.Name,
            Image:  c.Image,
        }

        result[serviceName] = append(result[serviceName], entry)
    }

    return result, runningImages
}

// extractServiceName extracts the service name from a Docker Compose container name.
// Format: stackname-servicename-N (e.g., "web-app-nginx-1" -> "nginx")
func extractServiceName(containerName string) string {
    parts := strings.Split(containerName, "-")
    if len(parts) < 3 {
        return containerName
    }
    // Remove the last part (instance number) and the stack name prefix
    // This is a best-effort heuristic; the Service field is preferred
    return parts[len(parts)-2]
}

// handleDockerStats returns resource usage stats via the Docker client.
// Args: [stackName] — if provided, only fetches stats for that stack's containers.
func (app *App) handleDockerStats(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    args := parseArgs(msg)
    stackName := argString(args, 0)

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    stats, err := app.Docker.ContainerStats(ctx, stackName)
    if err != nil {
        slog.Warn("dockerStats", "err", err)
        stats = map[string]docker.ContainerStat{}
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, struct {
            OK          bool                           `json:"ok"`
            DockerStats map[string]docker.ContainerStat `json:"dockerStats"`
        }{
            OK:          true,
            DockerStats: stats,
        })
    }
}

// handleContainerInspect returns full container inspect data via the Docker client.
func (app *App) handleContainerInspect(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    args := parseArgs(msg)
    containerName := argString(args, 0)
    if containerName == "" {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Container name required"})
        }
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    inspectData, err := app.Docker.ContainerInspect(ctx, containerName)
    if err != nil {
        slog.Warn("containerInspect", "err", err, "container", containerName)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
        }
        return
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, struct {
            OK          bool   `json:"ok"`
            InspectData string `json:"inspectData"`
        }{
            OK:          true,
            InspectData: inspectData,
        })
    }
}

// handleContainerTop returns running processes inside a container.
func (app *App) handleContainerTop(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    args := parseArgs(msg)
    containerName := argString(args, 0)
    if containerName == "" {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Container name required"})
        }
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    titles, processes, err := app.Docker.ContainerTop(ctx, containerName)
    if err != nil {
        slog.Warn("containerTop", "err", err, "container", containerName)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
        }
        return
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, struct {
            OK        bool       `json:"ok"`
            Titles    []string   `json:"titles"`
            Processes [][]string `json:"processes"`
        }{
            OK:        true,
            Titles:    titles,
            Processes: processes,
        })
    }
}

// handleGetDockerNetworkList returns Docker network summaries via the Docker client.
func (app *App) handleGetDockerNetworkList(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    networks, err := app.Docker.NetworkList(ctx)
    if err != nil {
        slog.Warn("getDockerNetworkList", "err", err)
        networks = []docker.NetworkSummary{}
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, struct {
            OK                bool                   `json:"ok"`
            DockerNetworkList []docker.NetworkSummary `json:"dockerNetworkList"`
        }{
            OK:                true,
            DockerNetworkList: networks,
        })
    }
}

// handleNetworkInspect returns detailed info for a single Docker network.
func (app *App) handleNetworkInspect(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    args := parseArgs(msg)
    networkName := argString(args, 0)
    if networkName == "" {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Network name required"})
        }
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    detail, err := app.Docker.NetworkInspect(ctx, networkName)
    if err != nil {
        slog.Warn("networkInspect", "err", err, "network", networkName)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
        }
        return
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, struct {
            OK            bool                `json:"ok"`
            NetworkDetail *docker.NetworkDetail `json:"networkDetail"`
        }{
            OK:            true,
            NetworkDetail: detail,
        })
    }
}

// handleGetDockerImageList returns Docker image summaries via the Docker client.
func (app *App) handleGetDockerImageList(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    images, err := app.Docker.ImageList(ctx)
    if err != nil {
        slog.Warn("getDockerImageList", "err", err)
        images = []docker.ImageSummary{}
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, struct {
            OK              bool                  `json:"ok"`
            DockerImageList []docker.ImageSummary  `json:"dockerImageList"`
        }{
            OK:              true,
            DockerImageList: images,
        })
    }
}

// handleImageInspect returns detailed info for a single Docker image.
func (app *App) handleImageInspect(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    args := parseArgs(msg)
    imageRef := argString(args, 0)
    if imageRef == "" {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Image reference required"})
        }
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    detail, err := app.Docker.ImageInspectDetail(ctx, imageRef)
    if err != nil {
        slog.Warn("imageInspect", "err", err, "image", imageRef)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
        }
        return
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, struct {
            OK          bool                `json:"ok"`
            ImageDetail *docker.ImageDetail `json:"imageDetail"`
        }{
            OK:          true,
            ImageDetail: detail,
        })
    }
}

// handleGetDockerVolumeList returns Docker volume summaries via the Docker client.
func (app *App) handleGetDockerVolumeList(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    volumes, err := app.Docker.VolumeList(ctx)
    if err != nil {
        slog.Warn("getDockerVolumeList", "err", err)
        volumes = []docker.VolumeSummary{}
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, struct {
            OK               bool                    `json:"ok"`
            DockerVolumeList []docker.VolumeSummary   `json:"dockerVolumeList"`
        }{
            OK:               true,
            DockerVolumeList: volumes,
        })
    }
}

// handleVolumeInspect returns detailed info for a single Docker volume.
func (app *App) handleVolumeInspect(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    args := parseArgs(msg)
    volumeName := argString(args, 0)
    if volumeName == "" {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Volume name required"})
        }
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    detail, err := app.Docker.VolumeInspect(ctx, volumeName)
    if err != nil {
        slog.Warn("volumeInspect", "err", err, "volume", volumeName)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
        }
        return
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, struct {
            OK           bool                  `json:"ok"`
            VolumeDetail *docker.VolumeDetail  `json:"volumeDetail"`
        }{
            OK:           true,
            VolumeDetail: detail,
        })
    }
}
