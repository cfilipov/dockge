package handlers

import (
    "bufio"
    "bytes"
    "context"
    "fmt"
    "log/slog"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "sync"
    "time"

    "github.com/cfilipov/dockge/internal/compose"
    "github.com/cfilipov/dockge/internal/terminal"
    "github.com/cfilipov/dockge/internal/ws"
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
        // Atomic join: register writer AND read buffer under a single lock.
        // This prevents a race where data arrives between separate Buffer()
        // and AddWriter() calls, causing duplicate delivery (double prompt).
        buf = term.JoinAndGetBuffer(c.ID(), makeTermWriter(c, termName))
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

    term := app.Terms.Recreate(termName, terminal.TypePTY)
    // Writer is carried over from the terminalJoin that already ran.
    // Do NOT add a duplicate writer — that causes the double-prompt race.

    dir := filepath.Join(app.StacksDir, stackName)
    execArgs := []string{"compose"}
    execArgs = append(execArgs, compose.GlobalEnvArgs(app.StacksDir, stackName)...)
    execArgs = append(execArgs, "exec", serviceName, shell)
    cmd := exec.Command("docker", execArgs...)
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

    term := app.Terms.Recreate(termName, terminal.TypePTY)
    // Writer is carried over from the terminalJoin that already ran.

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
            b := scanner.Bytes()
            term.Write(append(b, '\n'))
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
            b := scanner.Bytes()
            term.Write(append(b, '\n'))
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

// ANSI color palette for per-service log prefixes (6 high-contrast colors).
var logColors = [...]string{
    "\033[36m",  // cyan
    "\033[33m",  // yellow
    "\033[32m",  // green
    "\033[35m",  // magenta
    "\033[34m",  // blue
    "\033[91m",  // bright red
}

const colorReset = "\033[0m"

// coloredPrefix returns " serviceName | " with ANSI color, padded to maxLen.
func coloredPrefix(service string, maxLen int, colorIdx int) string {
    color := logColors[colorIdx%len(logColors)]
    return fmt.Sprintf("%s%-*s |%s ", color, maxLen, service, colorReset)
}

// runBanner returns a bold purple banner line marking a container start boundary.
// Returns empty string if startedAt is zero (mock mode / unknown).
func runBanner(service string, startedAt time.Time) string {
    if startedAt.IsZero() {
        return ""
    }
    ts := startedAt.Local().Format("15:04:05")
    // Bold, black text (38;2;0;0;0) on purple background (48;2;199;166;255) — #c7a6ff
    return fmt.Sprintf("\n\033[1;38;2;0;0;0;48;2;199;166;255m \u25B6 CONTAINER START \u2014 %s (%s) \033[0m\n\n",
        service, ts)
}

// startCombinedLogs creates a combined log terminal for a stack and starts
// streaming logs using per-container SDK log streams merged through a shared
// channel with a batched flusher. This replaces the previous `docker compose
// logs` subprocess approach, reducing memory from ~30MB to ~200KB per stack.
func (app *App) startCombinedLogs(termName, stackName string) *terminal.Terminal {
    term := app.Terms.Create(termName, terminal.TypePipe)

    ctx, cancel := context.WithCancel(context.Background())
    term.SetCancel(cancel)

    go app.runCombinedLogs(ctx, term, stackName)

    return term
}

// runCombinedLogs orchestrates per-container log readers and a batched flusher.
// It subscribes to Docker events to inject run-boundary banners on restarts and
// spawn new readers when containers are recreated or added. Blocks until ctx is
// cancelled.
func (app *App) runCombinedLogs(ctx context.Context, term *terminal.Terminal, stackName string) {
    containers, err := app.Docker.ContainerList(ctx, true, stackName)
    if err != nil {
        if ctx.Err() == nil {
            slog.Warn("combined logs: list containers", "err", err, "stack", stackName)
            term.Write([]byte("[Error] " + err.Error() + "\r\n"))
        }
        return
    }

    if len(containers) == 0 {
        return
    }

    // Compute max service name length for aligned prefixes
    maxLen := 0
    for _, c := range containers {
        if len(c.Service) > maxLen {
            maxLen = len(c.Service)
        }
    }

    // Assign stable color indices per service
    colorMap := make(map[string]int, len(containers))
    for i, c := range containers {
        colorMap[c.Service] = i
    }

    lineCh := make(chan []byte, 256)

    // Track which container IDs have an active reader goroutine.
    // Docker follow mode continues through restarts (same container ID),
    // so we only spawn new readers for genuinely new container IDs
    // (e.g. after docker compose up --force-recreate).
    var activeReaders sync.Map // containerID → struct{}

    // injectBanner fetches the container's start time and sends a banner.
    injectBanner := func(containerID, service string) {
        startedAt, err := app.Docker.ContainerStartedAt(ctx, containerID)
        if err != nil || startedAt.IsZero() {
            return
        }
        if banner := runBanner(service, startedAt); banner != "" {
            select {
            case lineCh <- []byte(banner):
            case <-ctx.Done():
            }
        }
    }

    // Spawn initial readers with banners (tail=100 for history)
    for _, c := range containers {
        injectBanner(c.ID, c.Service)
        activeReaders.Store(c.ID, struct{}{})
        go func(id, svc string, idx int) {
            defer activeReaders.Delete(id)
            app.readContainerLogs(ctx, id, svc, maxLen, idx, "100", lineCh)
        }(c.ID, c.Service, colorMap[c.Service])
    }

    // Watch for container start events:
    // - Always inject a banner (marks the restart boundary in the log stream)
    // - Only spawn a new reader if this container ID doesn't already have one
    //   (handles recreate where the old container is destroyed and a new one starts)
    // Event-spawned readers use tail="0" to avoid re-fetching historical lines
    // that the previous reader already streamed.
    eventCh, _ := app.Docker.Events(ctx)
    go func() {
        for {
            select {
            case evt, ok := <-eventCh:
                if !ok {
                    return
                }
                if evt.Project != stackName || evt.Action != "start" {
                    continue
                }
                idx, known := colorMap[evt.Service]
                if !known {
                    idx = len(colorMap)
                    colorMap[evt.Service] = idx
                    if len(evt.Service) > maxLen {
                        maxLen = len(evt.Service)
                    }
                }
                injectBanner(evt.ContainerID, evt.Service)
                // Spawn reader only for new container IDs (recreated containers)
                if _, loaded := activeReaders.LoadOrStore(evt.ContainerID, struct{}{}); !loaded {
                    go func(id, svc string, ci int) {
                        defer activeReaders.Delete(id)
                        app.readContainerLogs(ctx, id, svc, maxLen, ci, "0", lineCh)
                    }(evt.ContainerID, evt.Service, idx)
                }
            case <-ctx.Done():
                return
            }
        }
    }()

    // Flusher blocks until ctx is cancelled
    flushLogLines(ctx, term, lineCh)
}

// readContainerLogs streams logs for a single container, prefixing each line
// with a colored service name. Runs until the stream closes or ctx is cancelled.
// Banners are injected by the caller, not here. Use tail="100" for initial
// readers (show history) and tail="0" for event-spawned readers (follow only).
func (app *App) readContainerLogs(ctx context.Context, containerID, service string, maxLen, colorIdx int, tail string, lineCh chan<- []byte) {
    stream, _, err := app.Docker.ContainerLogs(ctx, containerID, tail, true)
    if err != nil {
        if ctx.Err() == nil {
            slog.Warn("combined logs: container stream", "err", err, "container", containerID)
        }
        return
    }
    defer stream.Close()

    prefix := coloredPrefix(service, maxLen, colorIdx)

    scanner := bufio.NewScanner(stream)
    scanner.Buffer(make([]byte, 64*1024), 64*1024)
    for scanner.Scan() {
        line := make([]byte, 0, len(prefix)+len(scanner.Bytes())+1)
        line = append(line, prefix...)
        line = append(line, scanner.Bytes()...)
        line = append(line, '\n')

        select {
        case lineCh <- line:
        case <-ctx.Done():
            return
        }
    }
}

// flushLogLines drains lineCh in batches on a 50ms tick and writes them to the
// terminal. This coalesces many small per-line writes into fewer, larger writes
// which reduces WebSocket message volume.
func flushLogLines(ctx context.Context, term *terminal.Terminal, lineCh <-chan []byte) {
    ticker := time.NewTicker(50 * time.Millisecond)
    defer ticker.Stop()

    var batch bytes.Buffer
    batch.Grow(4096)

    drain := func() bool {
        for {
            select {
            case line, ok := <-lineCh:
                if !ok {
                    return false
                }
                batch.Write(line)
            default:
                return true
            }
        }
    }

    flush := func() {
        if batch.Len() > 0 {
            term.Write(batch.Bytes())
            batch.Reset()
        }
    }

    for {
        select {
        case <-ctx.Done():
            drain()
            flush()
            return
        case <-ticker.C:
            if !drain() {
                flush()
                return
            }
            flush()
        }
    }
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
