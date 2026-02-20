package handlers

import (
    "context"
    "fmt"
    "io"
    "log/slog"
    "os"
    "path/filepath"
    "sync"
    "time"

    "github.com/cfilipov/dockge/backend-go/internal/stack"
    "github.com/cfilipov/dockge/backend-go/internal/ws"
)

// stackCache holds the in-memory stack list, refreshed by the background goroutine.
var (
    stackCacheMu sync.RWMutex
    stackCache   map[string]*stack.Stack
)

func RegisterStackHandlers(app *App) {
    app.WS.Handle("requestStackList", app.handleRequestStackList)
    app.WS.Handle("getStack", app.handleGetStack)
    app.WS.Handle("saveStack", app.handleSaveStack)
    app.WS.Handle("deployStack", app.handleDeployStack)
    app.WS.Handle("startStack", app.handleStartStack)
    app.WS.Handle("stopStack", app.handleStopStack)
    app.WS.Handle("restartStack", app.handleRestartStack)
    app.WS.Handle("downStack", app.handleDownStack)
    app.WS.Handle("updateStack", app.handleUpdateStack)
    app.WS.Handle("deleteStack", app.handleDeleteStack)
    app.WS.Handle("forceDeleteStack", app.handleForceDeleteStack)
}

// StartStackListBroadcaster starts a background goroutine that refreshes the stack list
// every 10 seconds and broadcasts it to all authenticated clients.
func (app *App) StartStackListBroadcaster(ctx context.Context) {
    // Initial refresh
    app.refreshStackCache()
    app.broadcastStackList()

    go func() {
        ticker := time.NewTicker(10 * time.Second)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                app.refreshStackCache()
                app.broadcastStackList()
            }
        }
    }()
}

func (app *App) refreshStackCache() {
    ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
    defer cancel()

    composeLs, err := app.Compose.Ls(ctx)
    if err != nil {
        slog.Warn("refresh stack cache: compose ls", "err", err)
        composeLs = nil
    }

    stacks := stack.GetStackList(app.StacksDir, composeLs)

    stackCacheMu.Lock()
    stackCache = stacks
    stackCacheMu.Unlock()
}

func (app *App) broadcastStackList() {
    stackCacheMu.RLock()
    stacks := stackCache
    stackCacheMu.RUnlock()

    if stacks == nil {
        return
    }

    listJSON := stack.BuildStackListJSON(stacks, "")
    app.WS.BroadcastAuthenticated("agent", "stackList", map[string]interface{}{
        "ok":        true,
        "stackList": listJSON,
    })
}

// TriggerStackListRefresh refreshes the cache and broadcasts immediately (after a mutation).
func (app *App) TriggerStackListRefresh() {
    go func() {
        // Small delay to let docker compose state settle
        time.Sleep(500 * time.Millisecond)
        app.refreshStackCache()
        app.broadcastStackList()
    }()
}

func (app *App) handleRequestStackList(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    // Read from cache — never block on docker
    stackCacheMu.RLock()
    stacks := stackCache
    stackCacheMu.RUnlock()

    listJSON := map[string]interface{}{}
    if stacks != nil {
        listJSON = stack.BuildStackListJSON(stacks, "")
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }

    // Also push via event (the frontend expects this)
    c.SendEvent("agent", "stackList", map[string]interface{}{
        "ok":        true,
        "stackList": listJSON,
    })
}

func (app *App) handleGetStack(c *ws.Conn, msg *ws.ClientMessage) {
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

    // Read from cache for status, then load full YAML from disk
    stackCacheMu.RLock()
    cached, exists := stackCache[stackName]
    stackCacheMu.RUnlock()

    s := &stack.Stack{Name: stackName}
    if exists {
        s.Status = cached.Status
        s.IsManagedByDockge = cached.IsManagedByDockge
        s.ComposeFileName = cached.ComposeFileName
        s.ComposeOverrideFileName = cached.ComposeOverrideFileName
    }

    // Load YAML content from disk (fast — local file I/O)
    s.LoadFromDisk(app.StacksDir)

    hostname := "localhost"
    if h, err := app.Settings.Get("primaryHostname"); err == nil && h != "" {
        hostname = h
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":    true,
            "stack": s.ToJSON("", hostname),
        })
    }
}

func (app *App) handleSaveStack(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    args := parseArgs(msg)
    stackName := argString(args, 0)
    composeYAML := argString(args, 1)
    composeENV := argString(args, 2)
    composeOverrideYAML := argString(args, 3)
    // isAdd := argBool(args, 4)

    if stackName == "" || composeYAML == "" {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack name and compose YAML required"})
        }
        return
    }

    s := &stack.Stack{
        Name:                stackName,
        ComposeYAML:         composeYAML,
        ComposeENV:          composeENV,
        ComposeOverrideYAML: composeOverrideYAML,
    }

    if err := s.SaveToDisk(app.StacksDir); err != nil {
        slog.Error("save stack", "err", err, "stack", stackName)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
        }
        return
    }

    app.TriggerStackListRefresh()

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true, Msg: "Saved"})
    }
}

func (app *App) handleDeployStack(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }

    args := parseArgs(msg)
    stackName := argString(args, 0)
    composeYAML := argString(args, 1)
    composeENV := argString(args, 2)
    composeOverrideYAML := argString(args, 3)
    // isAdd := argBool(args, 4)

    if stackName == "" || composeYAML == "" {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack name and compose YAML required"})
        }
        return
    }

    s := &stack.Stack{
        Name:                stackName,
        ComposeYAML:         composeYAML,
        ComposeENV:          composeENV,
        ComposeOverrideYAML: composeOverrideYAML,
    }

    if err := s.SaveToDisk(app.StacksDir); err != nil {
        slog.Error("deploy stack save", "err", err, "stack", stackName)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
        }
        return
    }

    // Return immediately — non-blocking
    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }

    // Spawn compose up in background, stream output to terminal
    go app.runComposeAction(stackName, "up", func(ctx context.Context, w io.Writer) error {
        return app.Compose.Up(ctx, stackName, w)
    })
}

func (app *App) handleStartStack(c *ws.Conn, msg *ws.ClientMessage) {
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

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }

    go app.runComposeAction(stackName, "up", func(ctx context.Context, w io.Writer) error {
        return app.Compose.Up(ctx, stackName, w)
    })
}

func (app *App) handleStopStack(c *ws.Conn, msg *ws.ClientMessage) {
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

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }

    go app.runComposeAction(stackName, "stop", func(ctx context.Context, w io.Writer) error {
        return app.Compose.Stop(ctx, stackName, w)
    })
}

func (app *App) handleRestartStack(c *ws.Conn, msg *ws.ClientMessage) {
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

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }

    go app.runComposeAction(stackName, "restart", func(ctx context.Context, w io.Writer) error {
        return app.Compose.Restart(ctx, stackName, w)
    })
}

func (app *App) handleDownStack(c *ws.Conn, msg *ws.ClientMessage) {
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

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }

    go app.runComposeAction(stackName, "down", func(ctx context.Context, w io.Writer) error {
        return app.Compose.Down(ctx, stackName, w)
    })
}

func (app *App) handleUpdateStack(c *ws.Conn, msg *ws.ClientMessage) {
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

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }

    go app.runComposeAction(stackName, "update", func(ctx context.Context, w io.Writer) error {
        return app.Compose.PullAndUp(ctx, stackName, w)
    })
}

func (app *App) handleDeleteStack(c *ws.Conn, msg *ws.ClientMessage) {
    if checkLogin(c, msg) == 0 {
        return
    }
    args := parseArgs(msg)
    stackName := argString(args, 0)

    var opts struct {
        DeleteStackFiles bool `json:"deleteStackFiles"`
    }
    argObject(args, 1, &opts)

    if stackName == "" {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack name required"})
        }
        return
    }

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }

    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
        defer cancel()

        // Down first
        app.Compose.Down(ctx, stackName, &discardWriter{})

        // Delete files if requested
        if opts.DeleteStackFiles {
            dir := filepath.Join(app.StacksDir, stackName)
            if err := os.RemoveAll(dir); err != nil {
                slog.Error("delete stack files", "err", err, "stack", stackName)
            }
        }

        app.TriggerStackListRefresh()
        slog.Info("stack deleted", "stack", stackName)
    }()
}

func (app *App) handleForceDeleteStack(c *ws.Conn, msg *ws.ClientMessage) {
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

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }

    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
        defer cancel()

        app.Compose.Down(ctx, stackName, &discardWriter{})

        dir := filepath.Join(app.StacksDir, stackName)
        if err := os.RemoveAll(dir); err != nil {
            slog.Error("force delete stack", "err", err, "stack", stackName)
        }

        app.TriggerStackListRefresh()
        slog.Info("stack force deleted", "stack", stackName)
    }()
}

// runComposeAction runs a compose command in the background, streaming output
// to a terminal that fans out to WebSocket clients.
//
// Terminal naming follows the frontend convention:
//   compose-{endpoint}-{stackName} → for local endpoint: compose--{stackName}
//
// Output goes to the terminal buffer; connected clients receive it via the
// terminal's fan-out mechanism (registered through terminalJoin).
func (app *App) runComposeAction(stackName, action string, fn func(ctx context.Context, w io.Writer) error) {
    termName := "compose--" + stackName
    term := app.Terms.Create(termName, 0) // TypePipe

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    // Write command display
    cmdDisplay := fmt.Sprintf("$ docker compose %s\r\n", action)
    term.Write([]byte(cmdDisplay))

    if err := fn(ctx, term); err != nil {
        errMsg := fmt.Sprintf("\r\n[Error] %s\r\n", err.Error())
        term.Write([]byte(errMsg))
        slog.Error("compose action", "action", action, "stack", stackName, "err", err)
    } else {
        term.Write([]byte("\r\n[Done]\r\n"))
    }

    // Refresh stack list after mutation
    app.TriggerStackListRefresh()
}

// discardWriter silently discards all output.
type discardWriter struct{}

func (d *discardWriter) Write(p []byte) (int, error) { return len(p), nil }
