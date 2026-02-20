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

    "github.com/cfilipov/dockge/backend-go/internal/docker"
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

// StartStackWatcher starts a background goroutine that:
// 1. Does an initial full container list to populate the cache
// 2. Subscribes to Docker Events to react to container lifecycle changes
// 3. Keeps a slow fallback ticker (60s) as a safety net
//
// This replaces the old StartStackListBroadcaster which polled every 10s.
func (app *App) StartStackWatcher(ctx context.Context) {
    // Initial full refresh
    app.refreshStackCache()
    app.broadcastStackList()

    // Subscribe to Docker events
    eventCh, errCh := app.Docker.Events(ctx)

    go func() {
        // Fallback ticker — full refresh every 60s as safety net
        ticker := time.NewTicker(60 * time.Second)
        defer ticker.Stop()

        // Debounce: batch events that arrive within 500ms into a single refresh
        var debounceTimer *time.Timer
        var debounceMu sync.Mutex
        triggerDebounced := func() {
            debounceMu.Lock()
            defer debounceMu.Unlock()
            if debounceTimer != nil {
                debounceTimer.Stop()
            }
            debounceTimer = time.AfterFunc(500*time.Millisecond, func() {
                app.refreshStackCache()
                app.broadcastStackList()
            })
        }

        for {
            select {
            case <-ctx.Done():
                debounceMu.Lock()
                if debounceTimer != nil {
                    debounceTimer.Stop()
                }
                debounceMu.Unlock()
                return

            case evt, ok := <-eventCh:
                if !ok {
                    // Event channel closed — fall back to polling only
                    slog.Warn("docker events channel closed, falling back to polling")
                    app.runPollingFallback(ctx)
                    return
                }
                slog.Debug("docker event", "action", evt.Action, "project", evt.Project, "service", evt.Service)
                triggerDebounced()

            case err, ok := <-errCh:
                if !ok {
                    continue
                }
                slog.Warn("docker events error", "err", err)
                // Reconnect: fall back to polling
                app.runPollingFallback(ctx)
                return

            case <-ticker.C:
                app.refreshStackCache()
                app.broadcastStackList()
            }
        }
    }()
}

// runPollingFallback runs a simple 60s polling loop when events are unavailable.
func (app *App) runPollingFallback(ctx context.Context) {
    ticker := time.NewTicker(60 * time.Second)
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
}

// refreshStackCache queries the Docker client for all compose containers,
// groups them by project, and merges with the filesystem scan.
func (app *App) refreshStackCache() {
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    // Get all containers with compose labels
    containers, err := app.Docker.ContainerList(ctx, true, "")
    if err != nil {
        slog.Warn("refresh stack cache: container list", "err", err)
    }

    stacks := stack.GetStackListFromContainers(app.StacksDir, containers)

    stackCacheMu.Lock()
    stackCache = stacks
    stackCacheMu.Unlock()

    // Compute recreateNecessary for all running stacks
    app.refreshRecreateCache(stacks, containers)
}

// refreshRecreateCache compares running images with compose.yaml images and
// populates the recreate cache. Uses the already-fetched container list.
func (app *App) refreshRecreateCache(stacks map[string]*stack.Stack, containers []docker.Container) {
    // Group containers by project
    byProject := make(map[string][]docker.Container)
    for _, c := range containers {
        if c.Project != "" {
            byProject[c.Project] = append(byProject[c.Project], c)
        }
    }

    for name, s := range stacks {
        if !s.IsStarted() {
            continue
        }
        projectContainers := byProject[name]
        if len(projectContainers) == 0 {
            continue
        }

        // Build running images map
        runningImages := make(map[string]string)
        for _, c := range projectContainers {
            if c.Service != "" {
                runningImages[c.Service] = c.Image
            }
        }

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
// In mock mode: uses pipe terminal + Composer interface (no real docker CLI).
// In real mode: uses PTY terminal + exec.Command (for rich terminal output).
func (app *App) runComposeAction(stackName, action string, composeArgs ...string) {
    termName := "compose--" + stackName
    cmdDisplay := fmt.Sprintf("$ docker compose %s\r\n", strings.Join(composeArgs, " "))

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    if app.Mock {
        term := app.Terms.Recreate(termName, terminal.TypePipe)
        term.Write([]byte(cmdDisplay))

        if err := app.Compose.RunCompose(ctx, stackName, term, composeArgs...); err != nil {
            if ctx.Err() == nil {
                errMsg := fmt.Sprintf("\r\n[Error] %s\r\n", err.Error())
                term.Write([]byte(errMsg))
                slog.Error("compose action", "action", action, "stack", stackName, "err", err)
            }
        } else {
            term.Write([]byte("\r\n[Done]\r\n"))
        }
    } else {
        term := app.Terms.Recreate(termName, terminal.TypePTY)
        term.Write([]byte(cmdDisplay))

        dir := filepath.Join(app.StacksDir, stackName)
        args := append([]string{"compose"}, composeArgs...)
        cmd := exec.CommandContext(ctx, "docker", args...)
        cmd.Dir = dir

        if err := term.RunPTY(cmd); err != nil {
            if ctx.Err() == nil {
                errMsg := fmt.Sprintf("\r\n[Error] %s\r\n", err.Error())
                term.Write([]byte(errMsg))
                slog.Error("compose action", "action", action, "stack", stackName, "err", err)
            }
        } else {
            term.Write([]byte("\r\n[Done]\r\n"))
        }
    }

    app.TriggerStackListRefresh()
}

// runDeployWithValidation validates the compose file via `docker compose config`
// and then runs `docker compose up -d --remove-orphans`.
func (app *App) runDeployWithValidation(stackName string) {
    termName := "compose--" + stackName

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    if app.Mock {
        term := app.Terms.Recreate(termName, terminal.TypePipe)

        // Step 1: Validate
        term.Write([]byte("$ docker compose config --dry-run\r\n"))
        if err := app.Compose.Config(ctx, stackName, term); err != nil {
            if ctx.Err() == nil {
                errMsg := fmt.Sprintf("\r\n[Error] Validation failed: %s\r\n", err.Error())
                term.Write([]byte(errMsg))
                slog.Warn("deploy validation failed", "stack", stackName, "err", err)
            }
            return
        }

        // Step 2: Deploy
        term.Write([]byte("$ docker compose up -d --remove-orphans\r\n"))
        if err := app.Compose.RunCompose(ctx, stackName, term, "up", "-d", "--remove-orphans"); err != nil {
            if ctx.Err() == nil {
                errMsg := fmt.Sprintf("\r\n[Error] %s\r\n", err.Error())
                term.Write([]byte(errMsg))
                slog.Error("compose action", "action", "deploy", "stack", stackName, "err", err)
            }
        } else {
            term.Write([]byte("\r\n[Done]\r\n"))
        }
    } else {
        term := app.Terms.Recreate(termName, terminal.TypePTY)
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
    }

    app.TriggerStackListRefresh()
}

// runDockerCommands runs multiple docker commands sequentially on the same terminal.
func (app *App) runDockerCommands(stackName, action string, argSets [][]string) {
    termName := "compose--" + stackName

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    if app.Mock {
        term := app.Terms.Recreate(termName, terminal.TypePipe)

        for _, dockerArgs := range argSets {
            cmdDisplay := fmt.Sprintf("$ docker %s\r\n", strings.Join(dockerArgs, " "))
            term.Write([]byte(cmdDisplay))

            var err error
            if len(dockerArgs) > 0 && dockerArgs[0] == "compose" {
                err = app.Compose.RunCompose(ctx, stackName, term, dockerArgs[1:]...)
            } else {
                err = app.Compose.RunDocker(ctx, stackName, term, dockerArgs...)
            }

            if err != nil {
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
    } else {
        term := app.Terms.Recreate(termName, terminal.TypePTY)
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
    }

    app.TriggerStackListRefresh()
}

// discardWriter silently discards all output.
type discardWriter struct{}

func (d *discardWriter) Write(p []byte) (int, error) { return len(p), nil }
