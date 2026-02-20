package handlers

import (
    "context"
    "fmt"
    "log/slog"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "sync"
    "time"

    "github.com/cfilipov/dockge/backend-go/internal/stack"
    "github.com/cfilipov/dockge/backend-go/internal/terminal"
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
    app.WS.Handle("pauseStack", app.handlePauseStack)
    app.WS.Handle("resumeStack", app.handleResumeStack)
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

    // Compute recreateNecessary for all running stacks so the rocket icon
    // appears immediately in the stack list (not only after viewing each stack).
    app.refreshRecreateCache(ctx, stacks)
}

// refreshRecreateCache runs `docker compose ps` for each running stack, compares
// running images with compose.yaml images, and populates the recreate cache.
func (app *App) refreshRecreateCache(ctx context.Context, stacks map[string]*stack.Stack) {
    for name, s := range stacks {
        if !s.IsStarted() {
            continue
        }
        psJSON, err := app.Compose.Ps(ctx, name)
        if err != nil {
            continue
        }
        _, runningImages := parseComposePsWithImages(psJSON)
        composeImages := parseComposeImages(app.StacksDir, name)
        anyRecreate := false
        for svc, runningImage := range runningImages {
            composeImage, ok := composeImages[svc]
            if ok && runningImage != "" && composeImage != "" && runningImage != composeImage {
                anyRecreate = true
                break
            }
        }
        app.SetRecreateNecessary(name, anyRecreate)
    }
}

func (app *App) broadcastStackList() {
    stackCacheMu.RLock()
    stacks := stackCache
    stackCacheMu.RUnlock()

    if stacks == nil {
        return
    }

    updateMap := app.GetImageUpdateMap()
    recreateMap := app.GetRecreateCache()
    listJSON := stack.BuildStackListJSON(stacks, "", updateMap, recreateMap)
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

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }

    // If cache is empty, refresh synchronously so the first load is instant
    stackCacheMu.RLock()
    empty := stackCache == nil || len(stackCache) == 0
    stackCacheMu.RUnlock()
    if empty {
        app.refreshStackCache()
    }

    app.sendStackListTo(c)
}

// sendStackListTo sends the cached stack list to a single connection.
func (app *App) sendStackListTo(c *ws.Conn) {
    stackCacheMu.RLock()
    stacks := stackCache
    stackCacheMu.RUnlock()

    listJSON := map[string]interface{}{}
    if stacks != nil {
        updateMap := app.GetImageUpdateMap()
        recreateMap := app.GetRecreateCache()
        listJSON = stack.BuildStackListJSON(stacks, "", updateMap, recreateMap)
    }

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

    updateMap := app.GetImageUpdateMap()
    recreateMap := app.GetRecreateCache()

    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":    true,
            "stack": s.ToJSON("", hostname, updateMap[stackName], recreateMap[stackName]),
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

    // Return immediately — validation and deploy run in background
    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }

    // Validate then deploy in background; errors stream to the terminal
    go app.runDeployWithValidation(stackName)
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

    go app.runComposeAction(stackName, "up", "up", "-d", "--remove-orphans")
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

    go app.runComposeAction(stackName, "stop", "stop")
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

    go app.runComposeAction(stackName, "restart", "restart")
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

    go app.runComposeAction(stackName, "down", "down")
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

    go app.runDockerCommands(stackName, "update", [][]string{
        {"compose", "pull"},
        {"compose", "up", "-d", "--remove-orphans"},
        {"image", "prune", "--all", "--force"},
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

        // Down first (with --remove-orphans, matching Node.js)
        app.Compose.DownRemoveOrphans(ctx, stackName, &discardWriter{})

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

        app.Compose.DownVolumes(ctx, stackName, &discardWriter{})

        dir := filepath.Join(app.StacksDir, stackName)
        if err := os.RemoveAll(dir); err != nil {
            slog.Error("force delete stack", "err", err, "stack", stackName)
        }

        app.TriggerStackListRefresh()
        slog.Info("stack force deleted", "stack", stackName)
    }()
}

func (app *App) handlePauseStack(c *ws.Conn, msg *ws.ClientMessage) {
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

    go app.runComposeAction(stackName, "pause", "pause")
}

func (app *App) handleResumeStack(c *ws.Conn, msg *ws.ClientMessage) {
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

    go app.runComposeAction(stackName, "unpause", "unpause")
}

// runComposeAction runs a compose command in the background, streaming output
// to a terminal that fans out to WebSocket clients.
//
// Terminal naming follows the frontend convention:
//
//	compose-{endpoint}-{stackName} → for local endpoint: compose--{stackName}
//
// The command runs through a PTY so that docker compose outputs colored,
// animated progress (it detects the TTY and enables rich output).
func (app *App) runComposeAction(stackName, action string, composeArgs ...string) {
    termName := "compose--" + stackName
    term := app.Terms.Recreate(termName, terminal.TypePTY)

    // Write command display
    cmdDisplay := fmt.Sprintf("$ docker compose %s\r\n", strings.Join(composeArgs, " "))
    term.Write([]byte(cmdDisplay))

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    dir := filepath.Join(app.StacksDir, stackName)
    args := append([]string{"compose"}, composeArgs...)
    cmd := exec.CommandContext(ctx, "docker", args...)
    cmd.Dir = dir

    if err := term.RunPTY(cmd); err != nil {
        // Only log if the context wasn't cancelled (user-initiated cancel is normal)
        if ctx.Err() == nil {
            errMsg := fmt.Sprintf("\r\n[Error] %s\r\n", err.Error())
            term.Write([]byte(errMsg))
            slog.Error("compose action", "action", action, "stack", stackName, "err", err)
        }
    } else {
        term.Write([]byte("\r\n[Done]\r\n"))
    }

    // Refresh stack list after mutation
    app.TriggerStackListRefresh()
}

// runDeployWithValidation validates the compose file via `docker compose config`
// and then runs `docker compose up -d --remove-orphans`. Both steps stream output
// to the terminal so the user sees any validation errors inline.
func (app *App) runDeployWithValidation(stackName string) {
    termName := "compose--" + stackName
    term := app.Terms.Recreate(termName, terminal.TypePTY)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    dir := filepath.Join(app.StacksDir, stackName)

    // Step 1: Validate
    term.Write([]byte("$ docker compose config --dry-run\r\n"))
    validateCmd := exec.CommandContext(ctx, "docker", "compose", "config", "--dry-run")
    validateCmd.Dir = dir
    if err := term.RunPTY(validateCmd); err != nil {
        if ctx.Err() == nil {
            errMsg := fmt.Sprintf("\r\n[Error] Validation failed: %s\r\n", err.Error())
            term.Write([]byte(errMsg))
            slog.Warn("deploy validation failed", "stack", stackName, "err", err)
        }
        return
    }

    // Step 2: Deploy
    term.Write([]byte("$ docker compose up -d --remove-orphans\r\n"))
    upCmd := exec.CommandContext(ctx, "docker", "compose", "up", "-d", "--remove-orphans")
    upCmd.Dir = dir
    if err := term.RunPTY(upCmd); err != nil {
        if ctx.Err() == nil {
            errMsg := fmt.Sprintf("\r\n[Error] %s\r\n", err.Error())
            term.Write([]byte(errMsg))
            slog.Error("compose action", "action", "deploy", "stack", stackName, "err", err)
        }
    } else {
        term.Write([]byte("\r\n[Done]\r\n"))
    }

    app.TriggerStackListRefresh()
}

// runDockerCommands runs multiple docker commands sequentially on the same PTY
// terminal. Each argSet is passed directly to `docker` (e.g., {"compose", "pull"}
// or {"image", "prune", "--all", "--force"}).
func (app *App) runDockerCommands(stackName, action string, argSets [][]string) {
    termName := "compose--" + stackName
    term := app.Terms.Recreate(termName, terminal.TypePTY)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    dir := filepath.Join(app.StacksDir, stackName)

    for _, dockerArgs := range argSets {
        cmdDisplay := fmt.Sprintf("$ docker %s\r\n", strings.Join(dockerArgs, " "))
        term.Write([]byte(cmdDisplay))

        cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
        cmd.Dir = dir

        if err := term.RunPTY(cmd); err != nil {
            if ctx.Err() == nil {
                errMsg := fmt.Sprintf("\r\n[Error] %s\r\n", err.Error())
                term.Write([]byte(errMsg))
                slog.Error("compose action", "action", action, "stack", stackName, "err", err)
            }
            app.TriggerStackListRefresh()
            return
        }
    }

    term.Write([]byte("\r\n[Done]\r\n"))
    app.TriggerStackListRefresh()
}

// discardWriter silently discards all output.
type discardWriter struct{}

func (d *discardWriter) Write(p []byte) (int, error) { return len(p), nil }
