package handlers

import (
    "github.com/cfilipov/dockge/backend-go/internal/ws"
)

func RegisterDockerHandlers(app *App) {
    app.WS.Handle("serviceStatusList", app.handleServiceStatusList)
    app.WS.Handle("dockerStats", app.handleDockerStats)
    app.WS.Handle("containerInspect", app.handleContainerInspect)
    app.WS.Handle("getDockerNetworkList", app.handleGetDockerNetworkList)
}

func (app *App) handleServiceStatusList(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }
    // TODO: Phase 5 — parse docker compose ps output per-service
    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":                    true,
            "serviceStatusList":     map[string]interface{}{},
            "serviceUpdateStatus":   map[string]interface{}{},
            "serviceRecreateStatus": map[string]interface{}{},
        })
    }
}

func (app *App) handleDockerStats(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }
    // TODO: Phase 5 — parse docker stats output
    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":   true,
            "data": map[string]interface{}{},
        })
    }
}

func (app *App) handleContainerInspect(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }
    // TODO: Phase 5 — docker inspect
    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":   true,
            "data": map[string]interface{}{},
        })
    }
}

func (app *App) handleGetDockerNetworkList(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }
    // TODO: Phase 5 — docker network ls
    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":          true,
            "networkList": []interface{}{},
        })
    }
}
