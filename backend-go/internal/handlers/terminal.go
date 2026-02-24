package handlers

import (
    "bufio"
    "context"
    "fmt"
    "log/slog"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "sync"
    "time"

    "github.com/cfilipov/dockge/backend-go/internal/terminal"
    "github.com/cfilipov/dockge/backend-go/internal/ws"
)

// ANSI color codes matching docker compose's service name palette.
var composeColors = []string{
    "\033[36m", // cyan
    "\033[33m", // yellow
    "\033[32m", // green
    "\033[35m", // magenta
    "\033[34m", // blue
    "\033[96m", // bright cyan
    "\033[93m", // bright yellow
    "\033[92m", // bright green
    "\033[95m", // bright magenta
    "\033[94m", // bright blue
}

const ansiReset = "\033[0m"

// coloredPrefix returns "serviceName | " with ANSI color, padded to align pipes.
func coloredPrefix(svcName string, colorIdx int, maxLen int) string {
    color := composeColors[colorIdx%len(composeColors)]
    padded := fmt.Sprintf("%-*s", maxLen, svcName)
    return color + padded + " | " + ansiReset
}

// mainTerminalMu guards mainTerminalName.
var mainTerminalMu sync.Mutex

func RegisterTerminalHandlers(app *App) {
    app.WS.Handle("terminalJoin", app.handleTerminalJoin)
    app.WS.Handle("terminalInput", app.handleTerminalInput)
    app.WS.Handle("terminalResize", app.handleTerminalResize)
    app.WS.Handle("mainTerminal", app.handleMainTerminal)
    app.WS.Handle("checkMainTerminal", app.handleCheckMainTerminal)
    app.WS.Handle("interactiveTerminal", app.handleInteractiveTerminal)
    app.WS.Handle("containerExec", app.handleContainerExec)
    app.WS.Handle("joinContainerLog", app.handleJoinContainerLog)
    app.WS.Handle("joinContainerLogByName", app.handleJoinContainerLogByName)
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
        c.SendAck(*msg.ID, struct {
            OK     bool   `json:"ok"`
            Buffer string `json:"buffer"`
        }{
            OK:     true,
            Buffer: buf,
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

// handleCheckMainTerminal checks if the main terminal is available and running.
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
        c.SendAck(*msg.ID, struct {
            OK        bool `json:"ok"`
            IsRunning bool `json:"isRunning"`
        }{
            OK:        true,
            IsRunning: running,
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

    // Schedule cleanup when the exec process exits
    term.OnExit(func() {
        app.Terms.RemoveAfter(termName, 30*time.Second)
    })

    slog.Info("interactive terminal started", "name", termName, "stack", stackName, "service", serviceName)

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }
}

// handleContainerExec creates a docker exec PTY terminal using the container name directly.
// Unlike handleInteractiveTerminal which uses docker compose exec, this uses docker exec
// and takes just the container name (e.g. "web-app-nginx-1") as the identifier.
func (app *App) handleContainerExec(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    args := parseArgs(msg)
    containerName := argString(args, 0)
    shell := argString(args, 1)

    if containerName == "" {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Container name required"})
        }
        return
    }
    if shell == "" {
        shell = "bash"
    }

    termName := "container-exec-by-name--" + containerName

    // Check if already running — register this client but don't recreate
    existing := app.Terms.Get(termName)
    if existing != nil && existing.IsRunning() {
        existing.AddWriter(c.ID(), makeTermWriter(c, termName))
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.OkResponse{OK: true})
        }
        return
    }

    term := app.Terms.Create(termName, terminal.TypePTY)

    // Register the requesting client BEFORE starting exec so the shell
    // prompt is captured and delivered.
    term.AddWriter(c.ID(), makeTermWriter(c, termName))

    cmd := exec.Command("docker", "exec", "-it", containerName, shell)
    cmd.Env = os.Environ()

    if err := term.StartPTY(cmd); err != nil {
        slog.Error("container exec start", "err", err, "container", containerName)
        app.Terms.Remove(termName)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Failed to start terminal: " + err.Error()})
        }
        return
    }

    // Schedule cleanup when the exec process exits
    term.OnExit(func() {
        app.Terms.RemoveAfter(termName, 30*time.Second)
    })

    slog.Info("container exec started", "name", termName, "container", containerName)

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }
}

// handleJoinContainerLog starts streaming logs for a single service using the
// Docker client's ContainerLogs API (SDK or mock).
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

    // Always recreate: the frontend's Terminal component mounts before the
    // parent ContainerLog.vue, so terminalJoin has already created an empty
    // terminal. Recreate carries over the registered writer while starting a
    // fresh log stream.
    term := app.Terms.Recreate(termName, terminal.TypePipe)

    ctx, cancel := context.WithCancel(context.Background())
    term.SetCancel(cancel)

    // Find the container ID for this service in this stack
    go func() {
        defer app.Terms.RemoveAfter(termName, 30*time.Second)

        containerID, err := app.findContainerID(ctx, stackName, serviceName)
        if err != nil {
            slog.Warn("joinContainerLog: find container", "err", err, "stack", stackName, "service", serviceName)
            term.Write([]byte("[Error] Could not find container for " + serviceName + "\r\n"))
            return
        }

        stream, _, err := app.Docker.ContainerLogs(ctx, containerID, "100", true)
        if err != nil {
            if ctx.Err() == nil {
                slog.Warn("container log stream", "err", err, "stack", stackName, "service", serviceName)
                term.Write([]byte("[Error] " + err.Error() + "\r\n"))
            }
            return
        }
        defer stream.Close()

        // Pipe log stream into the terminal
        scanner := bufio.NewScanner(stream)
        scanner.Buffer(make([]byte, 64*1024), 64*1024)
        for scanner.Scan() {
            line := scanner.Text() + "\n"
            term.Write([]byte(line))
        }
    }()

    // Register client for live updates (may already be carried over from Recreate)
    term.AddWriter(c.ID(), makeTermWriter(c, termName))

    if msg.ID != nil {
        c.SendAck(*msg.ID, struct {
            OK     bool   `json:"ok"`
            Buffer string `json:"buffer"`
        }{
            OK:     true,
            Buffer: term.Buffer(),
        })
    }
}

// handleJoinContainerLogByName streams logs for a container identified by its
// Docker container name (e.g. "web-app-nginx-1") rather than stack+service.
func (app *App) handleJoinContainerLogByName(c *ws.Conn, msg *ws.ClientMessage) {
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

    termName := "container-log-by-name--" + containerName

    // Always recreate: the frontend's Terminal component mounts before the
    // parent page, so terminalJoin has already created an empty terminal.
    // Recreate carries over the registered writer while starting a fresh
    // log stream.
    term := app.Terms.Recreate(termName, terminal.TypePipe)

    ctx, cancel := context.WithCancel(context.Background())
    term.SetCancel(cancel)

    go func() {
        defer app.Terms.RemoveAfter(termName, 30*time.Second)

        // Docker accepts container name directly — no need to resolve an ID
        stream, _, err := app.Docker.ContainerLogs(ctx, containerName, "100", true)
        if err != nil {
            if ctx.Err() == nil {
                slog.Warn("container log by name", "err", err, "container", containerName)
                term.Write([]byte("[Error] " + err.Error() + "\r\n"))
            }
            return
        }
        defer stream.Close()

        scanner := bufio.NewScanner(stream)
        scanner.Buffer(make([]byte, 64*1024), 64*1024)
        for scanner.Scan() {
            line := scanner.Text() + "\n"
            term.Write([]byte(line))
        }
    }()

    term.AddWriter(c.ID(), makeTermWriter(c, termName))

    if msg.ID != nil {
        c.SendAck(*msg.ID, struct {
            OK     bool   `json:"ok"`
            Buffer string `json:"buffer"`
        }{
            OK:     true,
            Buffer: term.Buffer(),
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

// startCombinedLogs creates a combined log terminal for a stack and starts streaming
// logs from all containers in the project using the Docker client.
func (app *App) startCombinedLogs(termName, stackName string) *terminal.Terminal {
    term := app.Terms.Create(termName, terminal.TypePipe)

    ctx, cancel := context.WithCancel(context.Background())
    term.SetCancel(cancel)

    go func() {
        // Find all containers for this stack
        containers, err := app.Docker.ContainerList(ctx, true, stackName)
        if err != nil {
            slog.Warn("combined logs: list containers", "err", err, "stack", stackName)
            term.Write([]byte("[Error] Could not list containers: " + err.Error() + "\r\n"))
            return
        }

        if len(containers) == 0 {
            term.Write([]byte("[Info] No containers found for stack " + stackName + "\r\n"))
            return
        }

        // Compute max service name length for aligned pipe prefixes
        maxLen := 0
        for _, c := range containers {
            if len(c.Service) > maxLen {
                maxLen = len(c.Service)
            }
        }

        // Open a log stream for each container and merge into the terminal
        var wg sync.WaitGroup
        for i, c := range containers {
            wg.Add(1)
            go func(containerID, svcName string, colorIdx int) {
                defer wg.Done()

                stream, _, err := app.Docker.ContainerLogs(ctx, containerID, "100", true)
                if err != nil {
                    if ctx.Err() == nil {
                        slog.Warn("combined log stream", "err", err, "service", svcName)
                    }
                    return
                }
                defer stream.Close()

                prefix := coloredPrefix(svcName, colorIdx, maxLen)
                scanner := bufio.NewScanner(stream)
                scanner.Buffer(make([]byte, 64*1024), 64*1024)
                for scanner.Scan() {
                    line := prefix + scanner.Text() + "\n"
                    term.Write([]byte(line))
                }
            }(c.ID, c.Service, i)
        }
        wg.Wait()
    }()

    return term
}

// findContainerID resolves a stack+service name to a container ID by querying
// the Docker client.
func (app *App) findContainerID(ctx context.Context, stackName, serviceName string) (string, error) {
    containers, err := app.Docker.ContainerList(ctx, true, stackName)
    if err != nil {
        return "", err
    }
    for _, c := range containers {
        if c.Service == serviceName {
            return c.ID, nil
        }
    }
    // Fallback: use container name (for mock mode where labels may not be set)
    for _, c := range containers {
        if strings.Contains(c.Name, serviceName) {
            return c.ID, nil
        }
    }
    // Last resort: return the name as-is (docker CLI will resolve it)
    return stackName + "-" + serviceName + "-1", nil
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
