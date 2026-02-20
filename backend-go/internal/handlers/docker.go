package handlers

import (
    "bufio"
    "context"
    "log/slog"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/cfilipov/dockge/backend-go/internal/docker"
    "github.com/cfilipov/dockge/backend-go/internal/ws"
)

func RegisterDockerHandlers(app *App) {
    app.WS.Handle("serviceStatusList", app.handleServiceStatusList)
    app.WS.Handle("dockerStats", app.handleDockerStats)
    app.WS.Handle("containerInspect", app.handleContainerInspect)
    app.WS.Handle("getDockerNetworkList", app.handleGetDockerNetworkList)
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
            c.SendAck(*msg.ID, map[string]interface{}{
                "ok":                    true,
                "serviceStatusList":     map[string]interface{}{},
                "serviceUpdateStatus":   map[string]interface{}{},
                "serviceRecreateStatus": map[string]interface{}{},
            })
        }
        return
    }

    serviceStatusList, runningImages := containersToServiceStatus(containers)

    // Compare running images vs compose.yaml to compute recreateNecessary per service
    composeImages := parseComposeImages(app.StacksDir, stackName)
    serviceRecreateStatus := make(map[string]interface{})
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
    serviceUpdateStatus := make(map[string]interface{})
    if svcUpdates, err := app.ImageUpdates.ServiceUpdatesForStack(stackName); err == nil {
        for svc, hasUpdate := range svcUpdates {
            serviceUpdateStatus[svc] = hasUpdate
        }
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":                    true,
            "serviceStatusList":     serviceStatusList,
            "serviceUpdateStatus":   serviceUpdateStatus,
            "serviceRecreateStatus": serviceRecreateStatus,
        })
    }
}

// containersToServiceStatus converts a list of containers (from the Docker client)
// into the serviceStatusList map and a running-images map, matching the format
// the frontend expects.
func containersToServiceStatus(containers []docker.Container) (map[string]interface{}, map[string]string) {
    result := make(map[string]interface{})
    runningImages := make(map[string]string)

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

        entry := map[string]interface{}{
            "status": status,
            "name":   c.Name,
            "image":  c.Image,
        }

        if existing, ok := result[serviceName]; ok {
            result[serviceName] = append(existing.([]interface{}), entry)
        } else {
            result[serviceName] = []interface{}{entry}
        }
    }

    return result, runningImages
}

// parseComposeImages reads the compose.yaml for a stack and extracts service->image mappings.
// Uses simple line parsing (no full YAML library needed).
func parseComposeImages(stacksDir, stackName string) map[string]string {
    result := make(map[string]string)
    composeFile := filepath.Join(stacksDir, stackName, "compose.yaml")
    f, err := os.Open(composeFile)
    if err != nil {
        return result
    }
    defer f.Close()

    scanner := bufio.NewScanner(f)
    inServices := false
    currentService := ""
    for scanner.Scan() {
        line := scanner.Text()
        trimmed := strings.TrimRight(line, " \t")

        // Detect "services:" top-level key
        if trimmed == "services:" {
            inServices = true
            continue
        }
        if !inServices {
            continue
        }
        // Exit services block on next top-level key
        if len(trimmed) > 0 && trimmed[0] != ' ' && trimmed[0] != '#' {
            break
        }
        // Service name: exactly 2-space indent, ends with ":"
        if len(line) > 2 && line[0] == ' ' && line[1] == ' ' && line[2] != ' ' && strings.HasSuffix(trimmed, ":") {
            currentService = strings.TrimSpace(strings.TrimSuffix(trimmed, ":"))
            continue
        }
        // Image field: 4+ space indent
        if currentService != "" && strings.Contains(line, "image:") {
            parts := strings.SplitN(line, "image:", 2)
            if len(parts) == 2 {
                img := strings.TrimSpace(parts[1])
                if img != "" {
                    result[currentService] = img
                }
            }
        }
    }
    return result
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
func (app *App) handleDockerStats(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    stats, err := app.Docker.ContainerStats(ctx)
    if err != nil {
        slog.Warn("dockerStats", "err", err)
        stats = map[string]docker.ContainerStat{}
    }

    // Convert to map[string]interface{} for JSON serialization
    statsMap := make(map[string]interface{}, len(stats))
    for k, v := range stats {
        statsMap[k] = v
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":          true,
            "dockerStats": statsMap,
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
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":          true,
            "inspectData": inspectData,
        })
    }
}

// handleGetDockerNetworkList returns Docker network names via the Docker client.
func (app *App) handleGetDockerNetworkList(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    networks, err := app.Docker.NetworkList(ctx)
    if err != nil {
        slog.Warn("getDockerNetworkList", "err", err)
        networks = []string{}
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":          true,
            "networkList": networks,
        })
    }
}
