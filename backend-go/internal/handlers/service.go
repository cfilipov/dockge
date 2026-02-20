package handlers

import (
    "context"
    "log/slog"
    "time"

    "github.com/cfilipov/dockge/backend-go/internal/docker"
    "github.com/cfilipov/dockge/backend-go/internal/ws"
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
        app.TriggerStackListRefresh()
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
        app.TriggerStackListRefresh()
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
        app.TriggerStackListRefresh()
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
        app.TriggerStackListRefresh()
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

    // Run the check in the background so we don't block the socket
    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
        defer cancel()

        // Get service→image from compose.yaml
        images := parseComposeImages(app.StacksDir, stackName)
        if len(images) == 0 {
            slog.Warn("checkImageUpdates: no images found", "stack", stackName)
            return
        }

        anyUpdate := false
        for svc, imageRef := range images {
            localDigest := docker.ImageDigest(ctx, imageRef)
            remoteDigest := docker.ManifestDigest(ctx, imageRef)

            hasUpdate := false
            if localDigest != "" && remoteDigest != "" && localDigest != remoteDigest {
                hasUpdate = true
                anyUpdate = true
            }

            if err := app.ImageUpdates.Upsert(stackName, svc, imageRef, localDigest, remoteDigest, hasUpdate); err != nil {
                slog.Error("checkImageUpdates upsert", "err", err, "stack", stackName, "svc", svc)
            }
        }

        slog.Info("image update check complete", "stack", stackName, "anyUpdate", anyUpdate)

        // Refresh the stack list so update icons appear/disappear
        app.TriggerStackListRefresh()
    }()

    // Ack immediately — the check runs asynchronously
    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":      true,
            "updated": true,
        })
    }
}
