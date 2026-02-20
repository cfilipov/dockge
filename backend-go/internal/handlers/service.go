package handlers

import (
    "context"
    "log/slog"
    "strings"
    "sync"
    "time"

    "github.com/cfilipov/dockge/backend-go/internal/ws"
)

const (
    imageUpdateInterval = 6 * time.Hour
    imageCheckConcurrency = 3
)

func RegisterServiceHandlers(app *App) {
    app.WS.Handle("startService", app.handleStartService)
    app.WS.Handle("stopService", app.handleStopService)
    app.WS.Handle("restartService", app.handleRestartService)
    app.WS.Handle("updateService", app.handleUpdateService)
    app.WS.Handle("checkImageUpdates", app.handleCheckImageUpdates)
}

func (app *App) handleStartService(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }
    args := parseArgs(msg)
    stackName := argString(args, 0)
    serviceName := argString(args, 1)
    if stackName == "" || serviceName == "" {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack and service name required"})
        }
        return
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }

    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
        defer cancel()
        if err := app.Compose.ServiceUp(ctx, stackName, serviceName, &discardWriter{}); err != nil {
            slog.Error("start service", "err", err, "stack", stackName, "service", serviceName)
        }
        app.TriggerStackListRefresh(stackName)
    }()
}

func (app *App) handleStopService(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }
    args := parseArgs(msg)
    stackName := argString(args, 0)
    serviceName := argString(args, 1)
    if stackName == "" || serviceName == "" {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack and service name required"})
        }
        return
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }

    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
        defer cancel()
        if err := app.Compose.ServiceStop(ctx, stackName, serviceName, &discardWriter{}); err != nil {
            slog.Error("stop service", "err", err, "stack", stackName, "service", serviceName)
        }
        app.TriggerStackListRefresh(stackName)
    }()
}

func (app *App) handleRestartService(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }
    args := parseArgs(msg)
    stackName := argString(args, 0)
    serviceName := argString(args, 1)
    if stackName == "" || serviceName == "" {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack and service name required"})
        }
        return
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }

    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
        defer cancel()
        if err := app.Compose.ServiceRestart(ctx, stackName, serviceName, &discardWriter{}); err != nil {
            slog.Error("restart service", "err", err, "stack", stackName, "service", serviceName)
        }
        app.TriggerStackListRefresh(stackName)
    }()
}

func (app *App) handleUpdateService(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }
    args := parseArgs(msg)
    stackName := argString(args, 0)
    serviceName := argString(args, 1)
    if stackName == "" || serviceName == "" {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack and service name required"})
        }
        return
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }

    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
        defer cancel()
        if err := app.Compose.ServicePullAndUp(ctx, stackName, serviceName, &discardWriter{}); err != nil {
            slog.Error("update service", "err", err, "stack", stackName, "service", serviceName)
        }
        app.TriggerStackListRefresh(stackName)
    }()
}

func (app *App) handleCheckImageUpdates(c *ws.Conn, msg *ws.ClientMessage) {
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

    go func() {
        app.checkImageUpdatesForStack(stackName)
        app.TriggerStackListRefresh(stackName)
    }()

    // Ack immediately — the check runs asynchronously
    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":      true,
            "updated": true,
        })
    }
}

// checkImageUpdatesForStack checks all services in a single stack for image updates.
// Safe to call from any goroutine.
func (app *App) checkImageUpdatesForStack(stackName string) {
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    images := parseComposeImages(app.StacksDir, stackName)
    if len(images) == 0 {
        return
    }

    anyUpdate := false
    for svc, imageRef := range images {
        localDigest := imageDigest(ctx, app, imageRef)
        remoteDigest := manifestDigest(ctx, app, imageRef)

        slog.Debug("image digest comparison", "svc", svc, "image", imageRef, "local", localDigest, "remote", remoteDigest)

        hasUpdate := localDigest != "" && remoteDigest != "" && localDigest != remoteDigest
        if hasUpdate {
            anyUpdate = true
        }

        if err := app.ImageUpdates.Upsert(stackName, svc, imageRef, localDigest, remoteDigest, hasUpdate); err != nil {
            slog.Error("checkImageUpdates upsert", "err", err, "stack", stackName, "svc", svc)
        }
    }

    slog.Info("image update check complete", "stack", stackName, "anyUpdate", anyUpdate)
}

// imageDigest returns the local digest for an image using the Docker client.
func imageDigest(ctx context.Context, app *App, imageRef string) string {
    digests, err := app.Docker.ImageInspect(ctx, imageRef)
    if err != nil || len(digests) == 0 {
        return ""
    }
    // RepoDigests are in the form "repo@sha256:abc..."
    for _, d := range digests {
        if idx := strings.Index(d, "@"); idx >= 0 {
            return d[idx+1:]
        }
    }
    return digests[0]
}

// manifestDigest returns the remote (registry) digest for an image using the Docker client.
func manifestDigest(ctx context.Context, app *App, imageRef string) string {
    digest, err := app.Docker.DistributionInspect(ctx, imageRef)
    if err != nil {
        return ""
    }
    return digest
}

// StartImageUpdateChecker starts a background goroutine that periodically checks
// all stacks for image updates. Runs once on startup (after a short delay) and
// then every 6 hours. Checks are parallelized with a concurrency limit of 3.
func (app *App) StartImageUpdateChecker(ctx context.Context) {
    go func() {
        // Short delay on startup so the stack list loads first
        select {
        case <-ctx.Done():
            return
        case <-time.After(30 * time.Second):
        }

        app.checkAllImageUpdates()
        app.broadcastStackList()

        ticker := time.NewTicker(imageUpdateInterval)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                app.checkAllImageUpdates()
                app.broadcastStackList()
            }
        }
    }()
}

// checkAllImageUpdates iterates all stacks and checks each for image updates,
// with a concurrency limit to avoid saturating the Docker daemon / network.
func (app *App) checkAllImageUpdates() {
    stackCacheMu.RLock()
    stacks := stackCache
    stackCacheMu.RUnlock()

    if len(stacks) == 0 {
        return
    }

    slog.Info("background image update check starting", "stacks", len(stacks))

    sem := make(chan struct{}, imageCheckConcurrency)
    var wg sync.WaitGroup

    for name := range stacks {
        wg.Add(1)
        sem <- struct{}{}
        go func(stackName string) {
            defer wg.Done()
            defer func() { <-sem }()
            app.checkImageUpdatesForStack(stackName)
        }(name)
    }

    wg.Wait()
    slog.Info("background image update check complete")
}
