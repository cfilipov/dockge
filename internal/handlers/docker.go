package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/cfilipov/dockge/internal/compose"
	"github.com/cfilipov/dockge/internal/docker"
	"github.com/cfilipov/dockge/internal/ws"
)

func RegisterDockerHandlers(app *App) {
	app.statsSubs = make(map[string]*statsSubscription)
	app.topSubs = make(map[string]*topSubscription)

	app.WS.Handle("serviceStatusList", app.handleServiceStatusList)
	app.WS.Handle("subscribeStats", app.handleSubscribeStats)
	app.WS.Handle("unsubscribeStats", app.handleUnsubscribeStats)
	app.WS.Handle("subscribeTop", app.handleSubscribeTop)
	app.WS.Handle("unsubscribeTop", app.handleUnsubscribeTop)
	app.WS.Handle("containerInspect", app.handleContainerInspect)
	app.WS.Handle("getDockerNetworkList", app.handleGetDockerNetworkList)
	app.WS.Handle("networkInspect", app.handleNetworkInspect)
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
	OK                    bool                      `json:"ok"`
	ServiceStatusList     map[string][]ServiceEntry `json:"serviceStatusList"`
	ServiceUpdateStatus   map[string]bool           `json:"serviceUpdateStatus"`
	ServiceRecreateStatus map[string]bool           `json:"serviceRecreateStatus"`
}

// handleServiceStatusList returns per-service status by querying Docker directly.
func (app *App) handleServiceStatusList(c *ws.Conn, msg *ws.ClientMessage) {
	if checkLogin(c, msg) == 0 {
		return
	}

	args := parseArgs(msg)
	stackName := argString(args, 0)
	if stackName == "" {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack name required"})
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
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Failed to list containers: " + err.Error()})
		}
		return
	}

	serviceStatusList, runningImages := containersToServiceStatus(containers)

	// Parse compose file from disk to get expected images
	composeImages := make(map[string]string)
	if path := compose.FindComposeFile(app.StacksDir, stackName); path != "" {
		services := compose.ParseFile(path)
		for svc, sd := range services {
			if sd.Image != "" {
				composeImages[svc] = sd.Image
			}
		}
	}

	// Compare running images vs compose.yaml to compute recreateNecessary per service
	serviceRecreateStatus := make(map[string]bool, len(runningImages))
	for svc, runningImage := range runningImages {
		composeImage, ok := composeImages[svc]
		if ok && runningImage != "" && composeImage != "" && runningImage != composeImage {
			serviceRecreateStatus[svc] = true
		} else {
			serviceRecreateStatus[svc] = false
		}
	}

	// Per-service image update status from BBolt cache
	serviceUpdateStatus := make(map[string]bool)
	if svcUpdates, err := app.ImageUpdates.ServiceUpdatesForStack(stackName); err == nil {
		serviceUpdateStatus = svcUpdates
	}

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, serviceStatusResponse{
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

// handleContainerInspect returns full container inspect data via the Docker client.
func (app *App) handleContainerInspect(c *ws.Conn, msg *ws.ClientMessage) {
	if checkLogin(c, msg) == 0 {
		return
	}

	args := parseArgs(msg)
	containerName := argString(args, 0)
	if containerName == "" {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Container name required"})
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	inspectData, err := app.Docker.ContainerInspect(ctx, containerName)
	if err != nil {
		slog.Warn("containerInspect", "err", err, "container", containerName)
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
		}
		return
	}

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, struct {
			OK          bool   `json:"ok"`
			InspectData json.RawMessage `json:"inspectData"`
		}{
			OK:          true,
			InspectData: inspectData,
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
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Failed to list networks: " + err.Error()})
		}
		return
	}

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, struct {
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
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Network name required"})
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	detail, err := app.Docker.NetworkInspect(ctx, networkName)
	if err != nil {
		slog.Warn("networkInspect", "err", err, "network", networkName)
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
		}
		return
	}

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, struct {
			OK            bool                  `json:"ok"`
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
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Failed to list images: " + err.Error()})
		}
		return
	}

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, struct {
			OK              bool                 `json:"ok"`
			DockerImageList []docker.ImageSummary `json:"dockerImageList"`
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
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Image reference required"})
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	detail, err := app.Docker.ImageInspectDetail(ctx, imageRef)
	if err != nil {
		slog.Warn("imageInspect", "err", err, "image", imageRef)
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
		}
		return
	}

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, struct {
			OK          bool                 `json:"ok"`
			ImageDetail *docker.ImageDetail `json:"imageDetail"`
		}{
			OK:          true,
			ImageDetail: detail,
		})
	}
}

// handleSubscribeStats starts a background goroutine that streams Docker stats
// for a single container to the client. Cancels any existing subscription.
// Args: [containerName]
func (app *App) handleSubscribeStats(c *ws.Conn, msg *ws.ClientMessage) {
	if checkLogin(c, msg) == 0 {
		return
	}

	args := parseArgs(msg)
	containerName := argString(args, 0)

	// Cancel any existing subscription for this connection
	app.cancelStatsSub(c.ID())

	ctx, cancel := context.WithCancel(context.Background())

	app.statsSubsMu.Lock()
	app.statsSubs[c.ID()] = &statsSubscription{cancel: cancel, container: containerName}
	app.statsSubsMu.Unlock()

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true})
	}

	go app.streamStats(ctx, c, containerName)
}

// handleUnsubscribeStats stops the stats streaming goroutine for this connection.
func (app *App) handleUnsubscribeStats(c *ws.Conn, msg *ws.ClientMessage) {
	app.cancelStatsSub(c.ID())
	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true})
	}
}

// streamStats opens a single streaming stats connection for one container
// and pushes updates to the client, throttled to one push per 5 seconds.
func (app *App) streamStats(ctx context.Context, c *ws.Conn, containerName string) {
	defer app.removeStatsSub(c.ID())

	if containerName == "" {
		return
	}

	statsCh, err := app.Docker.ContainerStatStream(ctx, containerName)
	if err != nil {
		slog.Debug("streamStats open", "err", err, "container", containerName)
		// Notify client that stats stream failed
		ws.SendEvent(c, "dockerStatsError", ws.ErrorResponse{OK: false, Msg: "Stats unavailable for " + containerName})
		return
	}

	// Throttle: push at most once per 5 seconds
	var lastPush time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.Done():
			return
		case stat, ok := <-statsCh:
			if !ok {
				return // stream ended (container stopped or EOF)
			}

			now := time.Now()
			if !lastPush.IsZero() && now.Sub(lastPush) < 5*time.Second {
				continue // skip intermediate frames
			}
			lastPush = now

			ws.SendEvent(c, "dockerStats", struct {
				OK          bool                            `json:"ok"`
				DockerStats map[string]docker.ContainerStat `json:"dockerStats"`
			}{
				OK:          true,
				DockerStats: map[string]docker.ContainerStat{containerName: stat},
			})
		}
	}
}

// cancelStatsSub cancels the stats subscription for the given connection ID.
func (app *App) cancelStatsSub(connID string) {
	app.statsSubsMu.Lock()
	defer app.statsSubsMu.Unlock()
	if sub, ok := app.statsSubs[connID]; ok {
		sub.cancel()
		delete(app.statsSubs, connID)
	}
}

// removeStatsSub removes a subscription entry (called by the goroutine on exit).
func (app *App) removeStatsSub(connID string) {
	app.statsSubsMu.Lock()
	defer app.statsSubsMu.Unlock()
	delete(app.statsSubs, connID)
}

// CancelStatsSub is the exported version for use in disconnect callbacks.
func (app *App) CancelStatsSub(connID string) {
	app.cancelStatsSub(connID)
}

// handleSubscribeTop starts a background goroutine that polls Docker container top
// (process list) and pushes updates to the client every 10 seconds.
// Args: [containerName]
func (app *App) handleSubscribeTop(c *ws.Conn, msg *ws.ClientMessage) {
	if checkLogin(c, msg) == 0 {
		return
	}

	args := parseArgs(msg)
	containerName := argString(args, 0)

	// Cancel any existing subscription for this connection
	app.cancelTopSub(c.ID())

	ctx, cancel := context.WithCancel(context.Background())

	app.topSubsMu.Lock()
	app.topSubs[c.ID()] = &topSubscription{cancel: cancel, container: containerName}
	app.topSubsMu.Unlock()

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true})
	}

	go app.streamTop(ctx, c, containerName)
}

// handleUnsubscribeTop stops the top streaming goroutine for this connection.
func (app *App) handleUnsubscribeTop(c *ws.Conn, msg *ws.ClientMessage) {
	app.cancelTopSub(c.ID())
	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true})
	}
}

// streamTop polls Docker container top every 10 seconds and pushes process list
// updates to the client. Sends one snapshot immediately on subscribe.
func (app *App) streamTop(ctx context.Context, c *ws.Conn, containerName string) {
	defer app.removeTopSub(c.ID())

	if containerName == "" {
		return
	}

	// Push one snapshot immediately
	app.pushTop(ctx, c, containerName)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.Done():
			return
		case <-ticker.C:
			if !app.pushTop(ctx, c, containerName) {
				return // error (container not running, etc.) — exit gracefully
			}
		}
	}
}

// pushTop fetches container top and sends it to the client. Returns false if
// the goroutine should exit (error or container not running).
func (app *App) pushTop(ctx context.Context, c *ws.Conn, containerName string) bool {
	fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	titles, processes, err := app.Docker.ContainerTop(fetchCtx, containerName)
	if err != nil {
		slog.Debug("streamTop poll", "err", err, "container", containerName)
		// Send empty update so frontend clears the list
		ws.SendEvent(c, "containerTop", struct {
			OK        bool       `json:"ok"`
			Titles    []string   `json:"titles"`
			Processes [][]string `json:"processes"`
		}{
			OK:        true,
			Titles:    []string{},
			Processes: [][]string{},
		})
		return false
	}

	ws.SendEvent(c, "containerTop", struct {
		OK        bool       `json:"ok"`
		Titles    []string   `json:"titles"`
		Processes [][]string `json:"processes"`
	}{
		OK:        true,
		Titles:    titles,
		Processes: processes,
	})
	return true
}

// cancelTopSub cancels the top subscription for the given connection ID.
func (app *App) cancelTopSub(connID string) {
	app.topSubsMu.Lock()
	defer app.topSubsMu.Unlock()
	if sub, ok := app.topSubs[connID]; ok {
		sub.cancel()
		delete(app.topSubs, connID)
	}
}

// removeTopSub removes a top subscription entry (called by the goroutine on exit).
func (app *App) removeTopSub(connID string) {
	app.topSubsMu.Lock()
	defer app.topSubsMu.Unlock()
	delete(app.topSubs, connID)
}

// CancelTopSub is the exported version for use in disconnect callbacks.
func (app *App) CancelTopSub(connID string) {
	app.cancelTopSub(connID)
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
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Failed to list volumes: " + err.Error()})
		}
		return
	}

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, struct {
			OK               bool                  `json:"ok"`
			DockerVolumeList []docker.VolumeSummary `json:"dockerVolumeList"`
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
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Volume name required"})
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	detail, err := app.Docker.VolumeInspect(ctx, volumeName)
	if err != nil {
		slog.Warn("volumeInspect", "err", err, "volume", volumeName)
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
		}
		return
	}

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, struct {
			OK           bool                  `json:"ok"`
			VolumeDetail *docker.VolumeDetail `json:"volumeDetail"`
		}{
			OK:           true,
			VolumeDetail: detail,
		})
	}
}
