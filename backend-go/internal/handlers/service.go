package handlers

import (
    "context"
    "log/slog"
    "time"

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
    // TODO: Phase 5 — check image updates via registry
    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":      true,
            "updated": false,
        })
    }
}
