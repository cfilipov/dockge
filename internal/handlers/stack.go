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

	"github.com/cfilipov/dockge/internal/compose"
	"github.com/cfilipov/dockge/internal/docker"
	"github.com/cfilipov/dockge/internal/stack"
	"github.com/cfilipov/dockge/internal/terminal"
	"github.com/cfilipov/dockge/internal/ws"
)

// freshData holds all the data needed for a broadcast, queried fresh each time.
type freshData struct {
	stacks         map[string]*stack.Stack
	byProject      map[string][]docker.Container
	updateMap      map[string]bool
	serviceUpdates map[string]bool
	recreateMap    map[string]bool
	imagesByStack  map[string]map[string]string
}

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

// queryFreshData queries Docker, filesystem, and BoltDB to build a complete
// snapshot of all stack data. No caches — each call is a fresh query.
func (app *App) queryFreshData() *freshData {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Query Docker (~10-50ms)
	containers, err := app.Docker.ContainerList(ctx, true, "")
	if err != nil {
		slog.Warn("queryFreshData: container list", "err", err)
	}

	// 2. Parse YAML from disk for ignore labels + image refs (~5ms for 50 stacks)
	ignoreMap, imagesByStack := parseAllComposeData(app.StacksDir)

	// 3. Build stack list from containers + filesystem
	stacks := stack.GetStackListFromContainers(app.StacksDir, containers, ignoreMap)
	byProject := groupByProject(containers)

	// 4. Read image update results from BoltDB (~0.5ms, memory-mapped)
	updateMap, _ := app.ImageUpdates.StackHasUpdates()
	serviceUpdates, _ := app.ImageUpdates.AllServiceUpdates()

	// 5. Compute recreate inline (compare running images vs compose YAML)
	recreateMap := computeRecreateMap(stacks, byProject, imagesByStack)

	return &freshData{
		stacks:         stacks,
		byProject:      byProject,
		updateMap:      updateMap,
		serviceUpdates: serviceUpdates,
		recreateMap:    recreateMap,
		imagesByStack:  imagesByStack,
	}
}

// parseAllComposeData scans the stacks directory, parses each compose file,
// and returns both the ignore map and the images-by-stack map.
func parseAllComposeData(stacksDir string) (stack.IgnoreMap, map[string]map[string]string) {
	entries, err := os.ReadDir(stacksDir)
	if err != nil {
		return nil, nil
	}

	ignoreMap := make(stack.IgnoreMap)
	imagesByStack := make(map[string]map[string]string)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		path := compose.FindComposeFile(stacksDir, name)
		if path == "" {
			continue
		}

		services := compose.ParseFile(path)
		images := make(map[string]string)
		for svc, sd := range services {
			if sd.Image != "" {
				images[svc] = sd.Image
			}
			if sd.StatusIgnore {
				if ignoreMap[name] == nil {
					ignoreMap[name] = make(map[string]bool)
				}
				ignoreMap[name][svc] = true
			}
		}
		imagesByStack[name] = images
	}
	return ignoreMap, imagesByStack
}

// groupByProject groups containers by compose project.
// Standalone containers (no project) are grouped under "_standalone".
func groupByProject(containers []docker.Container) map[string][]docker.Container {
	byProject := make(map[string][]docker.Container, len(containers))
	for _, c := range containers {
		key := c.Project
		if key == "" {
			key = "_standalone"
		}
		byProject[key] = append(byProject[key], c)
	}
	return byProject
}

// computeRecreateMap compares running container images with compose.yaml images
// to determine which stacks need recreation.
func computeRecreateMap(stacks map[string]*stack.Stack, byProject map[string][]docker.Container, imagesByStack map[string]map[string]string) map[string]bool {
	result := make(map[string]bool)
	for name, s := range stacks {
		if !s.IsStarted() {
			continue
		}
		projectContainers := byProject[name]
		if len(projectContainers) == 0 {
			continue
		}
		composeImages := imagesByStack[name]
		if len(composeImages) == 0 {
			continue
		}

		for _, c := range projectContainers {
			svc := c.Service
			if svc == "" {
				continue
			}
			compImg, ok := composeImages[svc]
			if ok && c.Image != "" && compImg != "" && c.Image != compImg {
				result[name] = true
				break
			}
		}
	}
	return result
}

// StartStackWatcher starts a background goroutine that:
// 1. Does an initial broadcast with fresh data
// 2. Subscribes to Docker Events to react to container lifecycle changes
// 3. Keeps a slow fallback ticker (60s) as a safety net
func (app *App) StartStackWatcher(ctx context.Context) {
	// Initial broadcast
	app.BroadcastAll()

	// Subscribe to Docker events
	eventCh, errCh := app.Docker.Events(ctx)

	go func() {
		// Fallback ticker — full refresh every 60s as safety net
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		// Debounce: batch events that arrive within 500ms into a single refresh.
		var debounceTimer *time.Timer
		var debounceMu sync.Mutex

		triggerDebounced := func() {
			debounceMu.Lock()
			defer debounceMu.Unlock()
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(500*time.Millisecond, func() {
				app.BroadcastAll()
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
				app.BroadcastAll()
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
			app.BroadcastAll()
		}
	}
}

// stackListResponse is the typed response for the stackList event.
type stackListResponse struct {
	OK        bool                             `json:"ok"`
	StackList map[string]stack.StackSimpleJSON `json:"stackList"`
}

// containerListResponse is the typed response for the containerList event.
type containerListResponse struct {
	OK            bool                       `json:"ok"`
	ContainerList []stack.ContainerSimpleJSON `json:"containerList"`
}

// BroadcastAll queries fresh data and broadcasts both stack list and container list
// to all authenticated connections. Skips all work if no clients are connected.
func (app *App) BroadcastAll() {
	if !app.WS.HasAuthenticatedConns() {
		return
	}

	data := app.queryFreshData()

	stackJSON := stack.BuildStackListJSON(data.stacks, "", data.updateMap, data.recreateMap)
	app.WS.BroadcastAuthenticatedRaw("agent", "stackList", stackListResponse{
		OK:        true,
		StackList: stackJSON,
	})

	containerJSON := stack.BuildContainerListJSON(data.byProject, data.stacks, data.serviceUpdates, data.recreateMap, data.imagesByStack)
	app.WS.BroadcastAuthenticatedRaw("agent", "containerList", containerListResponse{
		OK:            true,
		ContainerList: containerJSON,
	})
}

// TriggerRefresh broadcasts fresh data after a short delay to let Docker state settle.
// Uses a debounce timer so rapid successive calls coalesce into a single broadcast.
func (app *App) TriggerRefresh() {
	app.refreshMu.Lock()
	defer app.refreshMu.Unlock()
	if app.refreshTimer != nil {
		app.refreshTimer.Stop()
	}
	app.refreshTimer = time.AfterFunc(500*time.Millisecond, func() {
		app.BroadcastAll()
	})
}

func (app *App) handleRequestContainerList(c *ws.Conn, msg *ws.ClientMessage) {
	if checkLogin(c, msg) == 0 {
		return
	}

	if msg.ID != nil {
		c.SendAck(*msg.ID, ws.OkResponse{OK: true})
	}

	app.sendContainerListTo(c)
}

// sendContainerListTo sends a fresh container list to a single connection.
func (app *App) sendContainerListTo(c *ws.Conn) {
	data := app.queryFreshData()

	containerJSON := stack.BuildContainerListJSON(data.byProject, data.stacks, data.serviceUpdates, data.recreateMap, data.imagesByStack)
	if containerJSON == nil {
		containerJSON = []stack.ContainerSimpleJSON{}
	}

	c.SendEvent("agent", "containerList", containerListResponse{
		OK:            true,
		ContainerList: containerJSON,
	})
}

func (app *App) handleRequestStackList(c *ws.Conn, msg *ws.ClientMessage) {
	if checkLogin(c, msg) == 0 {
		return
	}

	if msg.ID != nil {
		c.SendAck(*msg.ID, ws.OkResponse{OK: true})
	}

	app.sendStackListTo(c)
}

// sendStackListTo sends a fresh stack list to a single connection.
func (app *App) sendStackListTo(c *ws.Conn) {
	data := app.queryFreshData()

	listJSON := stack.BuildStackListJSON(data.stacks, "", data.updateMap, data.recreateMap)
	if listJSON == nil {
		listJSON = map[string]stack.StackSimpleJSON{}
	}

	c.SendEvent("agent", "stackList", stackListResponse{
		OK:        true,
		StackList: listJSON,
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Query Docker for this stack's containers to determine status
	containers, _ := app.Docker.ContainerList(ctx, true, stackName)

	// Parse compose file for ignore labels
	ignoreMap, imagesByStack := parseAllComposeData(app.StacksDir)

	// Build status from containers
	stacks := stack.GetStackListFromContainers(app.StacksDir, containers, ignoreMap)

	s := &stack.Stack{Name: stackName}
	if cached, exists := stacks[stackName]; exists {
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

	// Compute recreate for this stack
	byProject := groupByProject(containers)
	recreateMap := computeRecreateMap(stacks, byProject, imagesByStack)

	if msg.ID != nil {
		c.SendAck(*msg.ID, struct {
			OK    bool               `json:"ok"`
			Stack stack.StackFullJSON `json:"stack"`
		}{
			OK:    true,
			Stack: s.ToJSON("", hostname, updateMap[stackName], recreateMap[stackName]),
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

	// Handle imageupdates.check transitions
	app.handleComposeYAMLSave(stackName, composeYAML)

	app.TriggerRefresh()

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

	// Handle imageupdates.check transitions
	app.handleComposeYAMLSave(stackName, composeYAML)

	// Return immediately — validation and deploy run in background
	if msg.ID != nil {
		c.SendAck(*msg.ID, ws.OkResponse{OK: true, Msg: "Deployed"})
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
		c.SendAck(*msg.ID, ws.OkResponse{OK: true, Msg: "Started"})
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
		c.SendAck(*msg.ID, ws.OkResponse{OK: true, Msg: "Stopped"})
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
		c.SendAck(*msg.ID, ws.OkResponse{OK: true, Msg: "Restarted"})
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
		c.SendAck(*msg.ID, ws.OkResponse{OK: true, Msg: "Stopped"})
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
		c.SendAck(*msg.ID, ws.OkResponse{OK: true, Msg: "Updated"})
	}

	go func() {
		app.runDockerCommands(stackName, "update", [][]string{
			{"compose", "pull"},
			{"compose", "up", "-d", "--remove-orphans"},
		})
		// Prune dangling images via SDK (no docker CLI needed)
		if msg, err := app.Docker.ImagePrune(context.Background(), true); err != nil {
			slog.Warn("image prune after update", "stack", stackName, "err", err)
		} else {
			slog.Debug("image prune after update", "stack", stackName, "result", msg)
		}
		// Clear stale "update available" cache and re-check with new images
		if err := app.ImageUpdates.DeleteForStack(stackName); err != nil {
			slog.Warn("clear image update cache", "stack", stackName, "err", err)
		}
		app.checkImageUpdatesForStack(stackName)
		app.BroadcastAll()
	}()
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
		c.SendAck(*msg.ID, ws.OkResponse{OK: true, Msg: "Deleted"})
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

		app.TriggerRefresh()
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
		c.SendAck(*msg.ID, ws.OkResponse{OK: true, Msg: "Deleted"})
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		app.Compose.DownVolumes(ctx, stackName, &discardWriter{})

		dir := filepath.Join(app.StacksDir, stackName)
		if err := os.RemoveAll(dir); err != nil {
			slog.Error("force delete stack", "err", err, "stack", stackName)
		}

		app.TriggerRefresh()
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
		c.SendAck(*msg.ID, ws.OkResponse{OK: true, Msg: "Started"})
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
		c.SendAck(*msg.ID, ws.OkResponse{OK: true, Msg: "Started"})
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
	envArgs := compose.GlobalEnvArgs(app.StacksDir, stackName)
	displayParts := append(envArgs, composeArgs...)
	cmdDisplay := fmt.Sprintf("$ docker compose %s\r\n", strings.Join(displayParts, " "))

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
		args := []string{"compose"}
		args = append(args, envArgs...)
		args = append(args, composeArgs...)
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

	// Schedule terminal cleanup after a grace period
	app.Terms.RemoveAfter(termName, 30*time.Second)

	app.TriggerRefresh()
}

// runDeployWithValidation validates the compose file via `docker compose config`
// and then runs `docker compose up -d --remove-orphans`.
func (app *App) runDeployWithValidation(stackName string) {
	termName := "compose--" + stackName
	envArgs := compose.GlobalEnvArgs(app.StacksDir, stackName)
	envDisplay := ""
	if len(envArgs) > 0 {
		envDisplay = strings.Join(envArgs, " ") + " "
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if app.Mock {
		term := app.Terms.Recreate(termName, terminal.TypePipe)

		// Step 1: Validate
		term.Write([]byte(fmt.Sprintf("$ docker compose %sconfig --dry-run\r\n", envDisplay)))
		if err := app.Compose.Config(ctx, stackName, term); err != nil {
			if ctx.Err() == nil {
				errMsg := fmt.Sprintf("\r\n[Error] Validation failed: %s\r\n", err.Error())
				term.Write([]byte(errMsg))
				slog.Warn("deploy validation failed", "stack", stackName, "err", err)
			}
			return
		}

		// Step 2: Deploy
		term.Write([]byte(fmt.Sprintf("$ docker compose %sup -d --remove-orphans\r\n", envDisplay)))
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
		term.Write([]byte(fmt.Sprintf("$ docker compose %sconfig --dry-run\r\n", envDisplay)))
		validateArgs := []string{"compose"}
		validateArgs = append(validateArgs, envArgs...)
		validateArgs = append(validateArgs, "config", "--dry-run")
		validateCmd := exec.CommandContext(ctx, "docker", validateArgs...)
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
		term.Write([]byte(fmt.Sprintf("$ docker compose %sup -d --remove-orphans\r\n", envDisplay)))
		upArgs := []string{"compose"}
		upArgs = append(upArgs, envArgs...)
		upArgs = append(upArgs, "up", "-d", "--remove-orphans")
		upCmd := exec.CommandContext(ctx, "docker", upArgs...)
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

	// Schedule terminal cleanup after a grace period
	app.Terms.RemoveAfter(termName, 30*time.Second)

	app.TriggerRefresh()
}

// runDockerCommands runs multiple docker commands sequentially on the same terminal.
func (app *App) runDockerCommands(stackName, action string, argSets [][]string) {
	termName := "compose--" + stackName
	envArgs := compose.GlobalEnvArgs(app.StacksDir, stackName)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if app.Mock {
		term := app.Terms.Recreate(termName, terminal.TypePipe)

		for _, dockerArgs := range argSets {
			cmdDisplay := fmt.Sprintf("$ docker %s\r\n", strings.Join(composeEnvDisplay(dockerArgs, envArgs), " "))
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
				app.TriggerRefresh()
				return
			}
		}

		term.Write([]byte("\r\n[Done]\r\n"))
	} else {
		term := app.Terms.Recreate(termName, terminal.TypePTY)
		dir := filepath.Join(app.StacksDir, stackName)

		for _, dockerArgs := range argSets {
			cmdDisplay := fmt.Sprintf("$ docker %s\r\n", strings.Join(composeEnvDisplay(dockerArgs, envArgs), " "))
			term.Write([]byte(cmdDisplay))

			var cmd *exec.Cmd
			if len(dockerArgs) > 0 && dockerArgs[0] == "compose" && len(envArgs) > 0 {
				args := []string{"compose"}
				args = append(args, envArgs...)
				args = append(args, dockerArgs[1:]...)
				cmd = exec.CommandContext(ctx, "docker", args...)
			} else {
				cmd = exec.CommandContext(ctx, "docker", dockerArgs...)
			}
			cmd.Dir = dir

			if err := term.RunPTY(cmd); err != nil {
				if ctx.Err() == nil {
					errMsg := fmt.Sprintf("\r\n[Error] %s\r\n", err.Error())
					term.Write([]byte(errMsg))
					slog.Error("compose action", "action", action, "stack", stackName, "err", err)
				}
				app.TriggerRefresh()
				return
			}
		}

		term.Write([]byte("\r\n[Done]\r\n"))
	}

	// Schedule terminal cleanup after a grace period
	app.Terms.RemoveAfter(termName, 30*time.Second)

	app.TriggerRefresh()
}

// handleComposeYAMLSave handles side effects of saving compose YAML:
// - Services with imageupdates.check=false → delete stale BBolt entries
// - Services with imageupdates.check re-enabled → trigger async check
func (app *App) handleComposeYAMLSave(stackName, composeYAML string) {
	newServices := compose.ParseYAML(composeYAML)

	for svc, sd := range newServices {
		if !sd.ImageUpdatesCheck {
			// Check disabled → clear stale BBolt entry
			if err := app.ImageUpdates.DeleteService(stackName, svc); err != nil {
				slog.Warn("clear disabled service update", "err", err, "stack", stackName, "svc", svc)
			}
		}
	}
}

// composeEnvDisplay injects env-file args into a docker command display string.
// If dockerArgs starts with "compose" and envArgs is non-empty, the env args
// are spliced in after "compose". Otherwise returns dockerArgs unchanged.
func composeEnvDisplay(dockerArgs, envArgs []string) []string {
	if len(dockerArgs) == 0 || dockerArgs[0] != "compose" || len(envArgs) == 0 {
		return dockerArgs
	}
	out := make([]string, 0, len(dockerArgs)+len(envArgs))
	out = append(out, "compose")
	out = append(out, envArgs...)
	out = append(out, dockerArgs[1:]...)
	return out
}

// discardWriter silently discards all output.
type discardWriter struct{}

func (d *discardWriter) Write(p []byte) (int, error) { return len(p), nil }
