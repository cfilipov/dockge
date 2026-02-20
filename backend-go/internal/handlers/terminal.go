package handlers

import (
    "github.com/cfilipov/dockge/backend-go/internal/ws"
)

func RegisterTerminalHandlers(app *App) {
    app.WS.Handle("terminalJoin", app.handleTerminalJoin)
    app.WS.Handle("terminalInput", app.handleTerminalInput)
    app.WS.Handle("terminalResize", app.handleTerminalResize)
    app.WS.Handle("mainTerminal", app.handleMainTerminal)
    app.WS.Handle("checkMainTerminal", app.handleCheckMainTerminal)
    app.WS.Handle("interactiveTerminal", app.handleInteractiveTerminal)
    app.WS.Handle("joinContainerLog", app.handleJoinContainerLog)
    app.WS.Handle("leaveCombinedTerminal", app.handleLeaveCombinedTerminal)
}

func (app *App) handleTerminalJoin(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    args := parseArgs(msg)
    termName := argString(args, 0)

    term := app.Terms.Get(termName)
    buf := ""
    if term != nil {
        buf = term.Buffer()
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":     true,
            "buffer": buf,
        })
    }
}

func (app *App) handleTerminalInput(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }
    // TODO: Phase 4 — write input to interactive terminals
    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }
}

func (app *App) handleTerminalResize(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }
    // TODO: Phase 4 — resize interactive terminals
    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }
}

func (app *App) handleMainTerminal(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }
    // TODO: Phase 4 — create main bash terminal
    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Main terminal not yet implemented"})
    }
}

func (app *App) handleCheckMainTerminal(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }
    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":       true,
            "isRunning": false,
        })
    }
}

func (app *App) handleInteractiveTerminal(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }
    // TODO: Phase 4 — create interactive container shell
    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Interactive terminal not yet implemented"})
    }
}

func (app *App) handleJoinContainerLog(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }
    // TODO: Phase 4 — stream container logs
    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }
}

func (app *App) handleLeaveCombinedTerminal(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }
    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }
}
