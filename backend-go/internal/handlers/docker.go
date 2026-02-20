package handlers

import (
    "context"
    "encoding/json"
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
    app.WS.Handle("getDockerNetworkList", app.handleGetDockerNetworkList)
}

// handleServiceStatusList runs `docker compose ps --format json` for a stack
// and returns per-service status entries.
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

    psJSON, err := app.Compose.Ps(ctx, stackName)
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

    serviceStatusList := parseComposePs(psJSON)

    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":                    true,
            "serviceStatusList":     serviceStatusList,
            "serviceUpdateStatus":   map[string]interface{}{},
            "serviceRecreateStatus": map[string]interface{}{},
        })
    }
}

// parseComposePs parses the JSON output of `docker compose ps --format json`
// into a map of service name → array of status entries.
//
// The output format can be either:
// - A JSON array: [{...}, {...}]
// - One JSON object per line (NDJSON): {...}\n{...}
func parseComposePs(data []byte) map[string]interface{} {
    result := make(map[string]interface{})
    trimmed := strings.TrimSpace(string(data))
    if trimmed == "" || trimmed == "[]" {
        return result
    }

    var containers []map[string]interface{}

    // Try JSON array first
    if err := json.Unmarshal(data, &containers); err != nil {
        // Try NDJSON (one object per line)
        for _, line := range strings.Split(trimmed, "\n") {
            line = strings.TrimSpace(line)
            if line == "" {
                continue
            }
            var obj map[string]interface{}
            if err := json.Unmarshal([]byte(line), &obj); err == nil {
                containers = append(containers, obj)
            }
        }
    }

    for _, c := range containers {
        // Extract service name from the container name or Service field
        serviceName := ""
        if svc, ok := c["Service"].(string); ok {
            serviceName = svc
        } else if name, ok := c["Name"].(string); ok {
            // Parse service name from container name: stackname-service-N
            serviceName = extractServiceName(name)
        }
        if serviceName == "" {
            continue
        }

        // Determine status from Health or State
        status := "unknown"
        if health, ok := c["Health"].(string); ok && health != "" {
            status = strings.ToLower(health)
        } else if state, ok := c["State"].(string); ok && state != "" {
            status = strings.ToLower(state)
        }

        entry := map[string]interface{}{
            "status": status,
            "name":   c["Name"],
            "image":  c["Image"],
        }

        // Append to service's entry list
        if existing, ok := result[serviceName]; ok {
            result[serviceName] = append(existing.([]interface{}), entry)
        } else {
            result[serviceName] = []interface{}{entry}
        }
    }

    return result
}

// extractServiceName extracts the service name from a Docker Compose container name.
// Format: stackname-servicename-N (e.g., "web-app-nginx-1" → "nginx")
func extractServiceName(containerName string) string {
    parts := strings.Split(containerName, "-")
    if len(parts) < 3 {
        return containerName
    }
    // Remove the last part (instance number) and the stack name prefix
    // This is a best-effort heuristic; the Service field is preferred
    return parts[len(parts)-2]
}

// handleDockerStats runs `docker stats --no-stream --format json`.
func (app *App) handleDockerStats(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    stats, err := docker.Stats(ctx)
    if err != nil {
        slog.Warn("dockerStats", "err", err)
        stats = map[string]interface{}{}
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":          true,
            "dockerStats": stats,
        })
    }
}

// handleContainerInspect runs `docker inspect <containerName>`.
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

    inspectData, err := docker.Inspect(ctx, containerName)
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

// handleGetDockerNetworkList runs `docker network ls`.
func (app *App) handleGetDockerNetworkList(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    networks, err := docker.NetworkList(ctx)
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
