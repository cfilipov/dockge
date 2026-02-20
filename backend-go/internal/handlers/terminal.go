package handlers

import (
    "context"
    "log/slog"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "sync"

    "github.com/cfilipov/dockge/backend-go/internal/terminal"
    "github.com/cfilipov/dockge/backend-go/internal/ws"
)

// mainTerminalMu guards mainTerminalName.
var mainTerminalMu sync.Mutex

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

// handleTerminalJoin joins a client to an existing terminal, returning the
// buffered output and registering the client for live updates.
// If the terminal name starts with "combined-" and doesn't exist yet, a
// combined log stream is started lazily.
func (app *App) handleTerminalJoin(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    args := parseArgs(msg)
    termName := argString(args, 0)
    if termName == "" {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Terminal name required"})
        }
        return
    }

    var term *terminal.Terminal

    // Lazy-start combined log terminals
    if strings.HasPrefix(termName, "combined-") {
        term = app.Terms.Get(termName)
        if term == nil {
            stackName := extractCombinedStackName(termName)
            if stackName != "" {
                term = app.startCombinedLogs(termName, stackName)
            }
        }
    } else {
        // For compose action terminals (compose--*) and others, create the
        // terminal on join if it doesn't exist yet. This ensures the writer
        // is registered before the compose action's Recreate() call, which
        // will carry over the writer to the fresh terminal.
        term = app.Terms.GetOrCreate(termName)
    }

    buf := ""
    if term != nil {
        buf = term.Buffer()
        // Register this connection for live updates
        term.AddWriter(c.ID(), makeTermWriter(c, termName))
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":     true,
            "buffer": buf,
        })
    }
}

// handleTerminalInput writes input to a terminal's PTY stdin.
func (app *App) handleTerminalInput(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    args := parseArgs(msg)
    termName := argString(args, 0)
    input := argString(args, 1)

    term := app.Terms.Get(termName)
    if term == nil {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Terminal not found"})
        }
        return
    }

    if err := term.Input(input); err != nil {
        slog.Warn("terminal input", "err", err, "term", termName)
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }
}

// handleTerminalResize resizes a terminal's PTY.
func (app *App) handleTerminalResize(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    args := parseArgs(msg)
    termName := argString(args, 0)
    rows := argInt(args, 1)
    cols := argInt(args, 2)

    term := app.Terms.Get(termName)
    if term != nil && rows > 0 && cols > 0 {
        if err := term.Resize(uint16(rows), uint16(cols)); err != nil {
            slog.Warn("terminal resize", "err", err, "term", termName)
        }
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }
}

// handleMainTerminal creates a bash shell PTY terminal.
func (app *App) handleMainTerminal(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    args := parseArgs(msg)
    termName := argString(args, 0)
    if termName == "" {
        termName = "console"
    }

    // Check if already running — register this client but don't recreate
    existing := app.Terms.Get(termName)
    if existing != nil && existing.IsRunning() {
        existing.AddWriter(c.ID(), makeTermWriter(c, termName))
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.OkResponse{OK: true})
        }
        return
    }

    // Create new PTY terminal
    term := app.Terms.Create(termName, terminal.TypePTY)

    // Register the requesting client BEFORE starting bash so the prompt
    // is captured and delivered.
    term.AddWriter(c.ID(), makeTermWriter(c, termName))

    shell := "bash"
    if _, err := exec.LookPath("bash"); err != nil {
        shell = "sh"
    }
    cmd := exec.Command(shell)
    cmd.Env = os.Environ()
    cmd.Dir = app.StacksDir

    if err := term.StartPTY(cmd); err != nil {
        slog.Error("main terminal start", "err", err)
        app.Terms.Remove(termName)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Failed to start terminal: " + err.Error()})
        }
        return
    }

    // Track the main terminal name for checkMainTerminal
    mainTerminalMu.Lock()
    app.MainTerminalName = termName
    mainTerminalMu.Unlock()

    slog.Info("main terminal started", "name", termName)

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }
}

// handleCheckMainTerminal checks if the main terminal is running.
func (app *App) handleCheckMainTerminal(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    mainTerminalMu.Lock()
    name := app.MainTerminalName
    mainTerminalMu.Unlock()

    running := false
    if name != "" {
        term := app.Terms.Get(name)
        if term != nil {
            running = term.IsRunning()
        }
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":        true,
            "isRunning": running,
        })
    }
}

// handleInteractiveTerminal creates a docker compose exec PTY terminal.
func (app *App) handleInteractiveTerminal(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    args := parseArgs(msg)
    stackName := argString(args, 0)
    serviceName := argString(args, 1)
    shell := argString(args, 2)

    if stackName == "" || serviceName == "" {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack name and service name required"})
        }
        return
    }
    if shell == "" {
        shell = "bash"
    }

    // Terminal name matches frontend convention:
    // container-exec-{endpoint}-{stackName}-{serviceName}-0
    // For local endpoint (empty string), becomes: container-exec--{stackName}-{serviceName}-0
    termName := "container-exec--" + stackName + "-" + serviceName + "-0"

    term := app.Terms.Create(termName, terminal.TypePTY)

    // Register the requesting client BEFORE starting exec so the shell
    // prompt is captured and delivered.
    term.AddWriter(c.ID(), makeTermWriter(c, termName))

    dir := filepath.Join(app.StacksDir, stackName)
    cmd := exec.Command("docker", "compose", "exec", serviceName, shell)
    cmd.Dir = dir
    cmd.Env = os.Environ()

    if err := term.StartPTY(cmd); err != nil {
        slog.Error("interactive terminal start", "err", err, "stack", stackName, "service", serviceName)
        app.Terms.Remove(termName)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Failed to start terminal: " + err.Error()})
        }
        return
    }

    slog.Info("interactive terminal started", "name", termName, "stack", stackName, "service", serviceName)

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }
}

// handleJoinContainerLog starts streaming logs for a single service.
func (app *App) handleJoinContainerLog(c *ws.Conn, msg *ws.ClientMessage) {
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

    // Terminal name: container-log-{endpoint}-{serviceName}
    // (matches frontend getContainerLogName convention)
    termName := "container-log--" + serviceName

    term := app.Terms.Get(termName)
    if term == nil {
        term = app.Terms.GetOrCreate(termName)
        // Start log streaming in background
        ctx, cancel := context.WithCancel(context.Background())
        term.SetCancel(cancel)

        go func() {
            if err := app.Compose.ServiceLogs(ctx, stackName, serviceName, term); err != nil {
                if ctx.Err() == nil {
                    slog.Warn("container log stream ended", "err", err, "stack", stackName, "service", serviceName)
                }
            }
        }()
    }

    // Register client for live updates
    term.AddWriter(c.ID(), makeTermWriter(c, termName))

    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":     true,
            "buffer": term.Buffer(),
        })
    }
}

// handleLeaveCombinedTerminal removes the client from a combined log terminal.
// If no clients remain, the log stream is cancelled.
func (app *App) handleLeaveCombinedTerminal(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    args := parseArgs(msg)
    stackName := argString(args, 0)

    if stackName != "" {
        termName := "combined--" + stackName
        term := app.Terms.Get(termName)
        if term != nil {
            term.RemoveWriter(c.ID())
            // If no more writers, stop the log stream
            if term.WriterCount() == 0 {
                app.Terms.Remove(termName)
            }
        }
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }
}

// startCombinedLogs creates a combined log terminal for a stack and starts streaming.
func (app *App) startCombinedLogs(termName, stackName string) *terminal.Terminal {
    term := app.Terms.GetOrCreate(termName)

    ctx, cancel := context.WithCancel(context.Background())
    term.SetCancel(cancel)

    go func() {
        if err := app.Compose.Logs(ctx, stackName, term); err != nil {
            if ctx.Err() == nil {
                slog.Warn("combined log stream ended", "err", err, "stack", stackName)
            }
        }
    }()

    return term
}

// extractCombinedStackName extracts the stack name from a combined terminal name.
// Format: "combined-{endpoint}-{stackName}" — for local endpoint: "combined--{stackName}"
func extractCombinedStackName(termName string) string {
    // Strip "combined-" prefix
    rest := strings.TrimPrefix(termName, "combined-")
    // The first segment is the endpoint (possibly empty), then stack name
    idx := strings.Index(rest, "-")
    if idx < 0 {
        return rest
    }
    return rest[idx+1:]
}

// makeTermWriter creates a WriteFunc that sends terminalWrite events to a connection.
func makeTermWriter(c *ws.Conn, termName string) terminal.WriteFunc {
    return func(data string) {
        c.SendEvent("agent", "terminalWrite", termName, data)
    }
}
