package handlers

import (
    "encoding/json"
    "log/slog"

    "github.com/cfilipov/dockge/backend-go/internal/ws"
)

func RegisterAgentHandlers(app *App) {
    app.WS.Handle("addAgent", app.handleAddAgent)
    app.WS.Handle("removeAgent", app.handleRemoveAgent)
    app.WS.Handle("updateAgent", app.handleUpdateAgent)

    // The "agent" event is the multiplexer: it unwraps the real event name
    // and dispatches to the correct handler.
    app.WS.Handle("agent", app.handleAgentDispatch)
}

func (app *App) handleAddAgent(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    args := parseArgs(msg)
    var data struct {
        URL      string `json:"url"`
        Username string `json:"username"`
        Password string `json:"password"`
        Name     string `json:"name"`
    }
    if !argObject(args, 0, &data) || data.URL == "" {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Invalid arguments"})
        }
        return
    }

    agent, err := app.Agents.Add(data.URL, data.Username, data.Password, data.Name)
    if err != nil {
        slog.Error("add agent", "err", err)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Failed to add agent"})
        }
        return
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":    true,
            "msg":   "successAdded",
            "agent": agent,
        })
    }
}

func (app *App) handleRemoveAgent(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    args := parseArgs(msg)
    url := argString(args, 0)
    if url == "" {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "URL required"})
        }
        return
    }

    if err := app.Agents.Remove(url); err != nil {
        slog.Error("remove agent", "err", err)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Failed to remove agent"})
        }
        return
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }
}

func (app *App) handleUpdateAgent(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    args := parseArgs(msg)
    url := argString(args, 0)
    name := argString(args, 1)
    if url == "" {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "URL required"})
        }
        return
    }

    if err := app.Agents.UpdateName(url, name); err != nil {
        slog.Error("update agent", "err", err)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Failed to update agent"})
        }
        return
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }
}

// handleAgentDispatch unwraps the "agent" event envelope and dispatches to
// the real handler. Frontend sends: {"event":"agent","args":["endpoint","realEvent",...]}
func (app *App) handleAgentDispatch(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    args := parseArgs(msg)
    if len(args) < 2 {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Invalid agent event"})
        }
        return
    }

    endpoint := argString(args, 0)
    eventName := argString(args, 1)

    // For non-empty endpoints, we'd forward to a remote agent.
    // For now, only handle local (empty endpoint).
    if endpoint != "" {
        slog.Warn("remote agent not implemented", "endpoint", endpoint, "event", eventName)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Remote agents not yet supported"})
        }
        return
    }

    // Re-pack remaining args as the inner message's args
    var innerArgs json.RawMessage
    if len(args) > 2 {
        remaining := args[2:]
        innerArgs, _ = json.Marshal(remaining)
    } else {
        innerArgs = []byte("[]")
    }

    innerMsg := &ws.ClientMessage{
        ID:    msg.ID,
        Event: eventName,
        Args:  innerArgs,
    }

    // Look up the handler by inner event name
    app.WS.Dispatch(c, innerMsg)
}
