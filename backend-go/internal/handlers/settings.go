package handlers

import (
    "fmt"
    "log/slog"
    "os"
    "path/filepath"

    "github.com/cfilipov/dockge/backend-go/internal/ws"
)

func RegisterSettingsHandlers(app *App) {
    app.WS.Handle("getSettings", app.handleGetSettings)
    app.WS.Handle("setSettings", app.handleSetSettings)
    app.WS.Handle("disconnectOtherSocketClients", app.handleDisconnectOthers)
    app.WS.Handle("composerize", app.handleComposerize)

    // Uptime Kuma heartbeat stubs — not applicable to Dockge standalone
    app.WS.Handle("monitorImportantHeartbeatListCount", app.handleStubOk)
    app.WS.Handle("monitorImportantHeartbeatListPaged", app.handleStubOk)
}

func (app *App) handleGetSettings(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    settings, err := app.Settings.GetAll()
    if err != nil {
        slog.Error("get settings", "err", err)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Failed to load settings"})
        }
        return
    }

    // Filter out sensitive settings
    delete(settings, "jwtSecret")

    // globalENV is file-based, not stored in BoltDB
    globalEnvPath := filepath.Join(app.StacksDir, "global.env")
    if data, err := os.ReadFile(globalEnvPath); err == nil {
        settings["globalENV"] = string(data)
    } else {
        settings["globalENV"] = "# VARIABLE=value #comment"
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":   true,
            "data": settings,
        })
    }
}

func (app *App) handleSetSettings(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    args := parseArgs(msg)
    var data map[string]interface{}
    if !argObject(args, 0, &data) {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Invalid arguments"})
        }
        return
    }

    // currentPassword is args[1] but we skip validation for now
    // (settings changes don't require password re-entry in the Node.js backend either,
    //  except for disableAuth)

    // globalENV is file-based — write to disk, not BoltDB
    if raw, ok := data["globalENV"]; ok {
        content, _ := raw.(string)
        globalEnvPath := filepath.Join(app.StacksDir, "global.env")
        defaultContent := "# VARIABLE=value #comment"
        if content != "" && content != defaultContent {
            if err := os.WriteFile(globalEnvPath, []byte(content), 0644); err != nil {
                slog.Error("write global.env", "err", err)
            }
        } else {
            os.Remove(globalEnvPath)
        }
        delete(data, "globalENV")
    }

    for key, val := range data {
        // Don't allow overwriting jwtSecret via settings
        if key == "jwtSecret" {
            continue
        }
        strVal := ""
        switch v := val.(type) {
        case string:
            strVal = v
        case bool:
            if v {
                strVal = "1"
            } else {
                strVal = "0"
            }
        case float64:
            strVal = fmt.Sprintf("%v", v)
        default:
            continue
        }
        if err := app.Settings.Set(key, strVal); err != nil {
            slog.Error("set setting", "key", key, "err", err)
        }
    }

    app.Settings.InvalidateCache()

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true, Msg: "Saved"})
    }
}

func (app *App) handleDisconnectOthers(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    app.WS.DisconnectOthers(c)

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }
}

// handleStubOk silently acknowledges events that are not applicable.
func (app *App) handleStubOk(c *ws.Conn, msg *ws.ClientMessage) {
    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }
}

func (app *App) handleComposerize(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    // Stubbed — composerize is not yet implemented in Go
    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.ErrorResponse{
            OK:  false,
            Msg: "Composerize is not available. Use https://composerize.com to convert docker run commands.",
        })
    }
}
