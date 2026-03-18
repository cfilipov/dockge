package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfilipov/dockge/internal/compose"
	"github.com/cfilipov/dockge/internal/docker"
	"github.com/cfilipov/dockge/internal/stack"
	"github.com/cfilipov/dockge/internal/terminal"
	"github.com/cfilipov/dockge/internal/ws"
)

func RegisterStackHandlers(app *App) {
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

// parseComposeDataForStack parses compose data for a single stack,
// avoiding the cost of scanning all stacks in the directory.
func parseComposeDataForStack(stacksDir, stackName string) (stack.IgnoreMap, map[string]map[string]string) {
	ignoreMap := make(stack.IgnoreMap)
	imagesByStack := make(map[string]map[string]string)

	path := compose.FindComposeFile(stacksDir, stackName)
	if path == "" {
		return ignoreMap, imagesByStack
	}

	services := compose.ParseFile(path)
	images := make(map[string]string)
	for svc, sd := range services {
		if sd.Image != "" {
			images[svc] = sd.Image
		}
		if sd.StatusIgnore {
			if ignoreMap[stackName] == nil {
				ignoreMap[stackName] = make(map[string]bool)
			}
			ignoreMap[stackName][svc] = true
		}
	}
	imagesByStack[stackName] = images
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


func (app *App) handleGetStack(c *ws.Conn, msg *ws.ClientMessage) {
	if checkLogin(c, msg) == 0 {
		return
	}

	args := parseArgs(msg)
	stackName := argString(args, 0)
	if stackName == "" {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack name required"})
		}
		return
	}
	if err := stack.ValidateStackName(stackName); err != nil {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Query Docker for this stack's containers to determine status
	containers, _ := app.Docker.ContainerList(ctx, true, stackName)

	// Parse compose file for the requested stack only (not all stacks).
	ignoreMap, imagesByStack := parseComposeDataForStack(app.StacksDir, stackName)

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
		ws.SendAck(c, *msg.ID, struct {
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
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack name and compose YAML required"})
		}
		return
	}
	if err := stack.ValidateStackName(stackName); err != nil {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
		}
		return
	}

	app.StackLocks.Lock(stackName)
	defer app.StackLocks.Unlock(stackName)

	s := &stack.Stack{
		Name:                stackName,
		ComposeYAML:         composeYAML,
		ComposeENV:          composeENV,
		ComposeOverrideYAML: composeOverrideYAML,
	}

	if err := s.SaveToDisk(app.StacksDir); err != nil {
		slog.Error("save stack", "err", err, "stack", stackName)
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
		}
		return
	}

	// Handle imageupdates.check transitions
	app.handleComposeYAMLSave(stackName, composeYAML)

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true, Msg: "Saved"})
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
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack name and compose YAML required"})
		}
		return
	}
	if err := stack.ValidateStackName(stackName); err != nil {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
		}
		return
	}

	app.StackLocks.Lock(stackName)

	s := &stack.Stack{
		Name:                stackName,
		ComposeYAML:         composeYAML,
		ComposeENV:          composeENV,
		ComposeOverrideYAML: composeOverrideYAML,
	}

	if err := s.SaveToDisk(app.StacksDir); err != nil {
		app.StackLocks.Unlock(stackName)
		slog.Error("deploy stack save", "err", err, "stack", stackName)
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
		}
		return
	}

	// Handle imageupdates.check transitions
	app.handleComposeYAMLSave(stackName, composeYAML)

	// Validate then deploy in background; ack after completion so the
	// frontend stays on the current page showing progress output.
	go func() {
		defer app.StackLocks.Unlock(stackName)
		app.runDeployWithValidation(stackName)
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true, Msg: "Deployed"})
		}
	}()
}

func (app *App) handleStartStack(c *ws.Conn, msg *ws.ClientMessage) {
	if checkLogin(c, msg) == 0 {
		return
	}
	args := parseArgs(msg)
	stackName := argString(args, 0)
	if stackName == "" {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack name required"})
		}
		return
	}
	if err := stack.ValidateStackName(stackName); err != nil {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
		}
		return
	}

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true})
	}

	if app.isStackManaged(stackName) {
		go app.lockedRunComposeAction(stackName, "up", "up", "-d", "--remove-orphans")
	} else {
		// Unmanaged: use docker compose -p to start existing containers
		go app.lockedRunUnmanagedStackAction(stackName, "start", "start")
	}
}

func (app *App) handleStopStack(c *ws.Conn, msg *ws.ClientMessage) {
	if checkLogin(c, msg) == 0 {
		return
	}
	args := parseArgs(msg)
	stackName := argString(args, 0)
	if stackName == "" {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack name required"})
		}
		return
	}
	if err := stack.ValidateStackName(stackName); err != nil {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
		}
		return
	}

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true})
	}

	if app.isStackManaged(stackName) {
		go app.lockedRunComposeAction(stackName, "stop", "stop")
	} else {
		go app.lockedRunUnmanagedStackAction(stackName, "stop", "stop")
	}
}

func (app *App) handleRestartStack(c *ws.Conn, msg *ws.ClientMessage) {
	if checkLogin(c, msg) == 0 {
		return
	}
	args := parseArgs(msg)
	stackName := argString(args, 0)
	if stackName == "" {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack name required"})
		}
		return
	}
	if err := stack.ValidateStackName(stackName); err != nil {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
		}
		return
	}

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true})
	}

	if app.isStackManaged(stackName) {
		go app.lockedRunComposeAction(stackName, "restart", "restart")
	} else {
		go app.lockedRunUnmanagedStackAction(stackName, "restart", "restart")
	}
}

func (app *App) handleDownStack(c *ws.Conn, msg *ws.ClientMessage) {
	if checkLogin(c, msg) == 0 {
		return
	}
	args := parseArgs(msg)
	stackName := argString(args, 0)
	if stackName == "" {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack name required"})
		}
		return
	}
	if err := stack.ValidateStackName(stackName); err != nil {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
		}
		return
	}

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true})
	}

	if app.isStackManaged(stackName) {
		go app.lockedRunComposeAction(stackName, "down", "down")
	} else {
		go app.lockedRunUnmanagedStackAction(stackName, "down", "down")
	}
}

func (app *App) handleUpdateStack(c *ws.Conn, msg *ws.ClientMessage) {
	if checkLogin(c, msg) == 0 {
		return
	}
	args := parseArgs(msg)
	stackName := argString(args, 0)
	if stackName == "" {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack name required"})
		}
		return
	}
	if err := stack.ValidateStackName(stackName); err != nil {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
		}
		return
	}

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true})
	}

	go func() {
		app.StackLocks.Lock(stackName)
		defer app.StackLocks.Unlock(stackName)

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
		app.TriggerUpdatesBroadcast()
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
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack name required"})
		}
		return
	}
	if err := stack.ValidateStackName(stackName); err != nil {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
		}
		return
	}

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true})
	}

	go func() {
		app.StackLocks.Lock(stackName)
		defer app.StackLocks.Unlock(stackName)

		// Down via terminal manager so users see progress output
		app.runComposeAction(stackName, "down", "down", "--remove-orphans")

		// Delete files if requested
		if opts.DeleteStackFiles {
			dir := filepath.Join(app.StacksDir, stackName)
			if err := os.RemoveAll(dir); err != nil {
				slog.Error("delete stack files", "err", err, "stack", stackName)
			}
		}

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
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack name required"})
		}
		return
	}
	if err := stack.ValidateStackName(stackName); err != nil {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
		}
		return
	}

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true})
	}

	go func() {
		app.StackLocks.Lock(stackName)
		defer app.StackLocks.Unlock(stackName)

		// Down via terminal manager so users see progress output
		app.runComposeAction(stackName, "down", "down", "-v", "--remove-orphans")

		dir := filepath.Join(app.StacksDir, stackName)
		if err := os.RemoveAll(dir); err != nil {
			slog.Error("force delete stack", "err", err, "stack", stackName)
		}

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
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack name required"})
		}
		return
	}
	if err := stack.ValidateStackName(stackName); err != nil {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
		}
		return
	}

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true})
	}

	go app.lockedRunComposeAction(stackName, "pause", "pause")
}

func (app *App) handleResumeStack(c *ws.Conn, msg *ws.ClientMessage) {
	if checkLogin(c, msg) == 0 {
		return
	}
	args := parseArgs(msg)
	stackName := argString(args, 0)
	if stackName == "" {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack name required"})
		}
		return
	}
	if err := stack.ValidateStackName(stackName); err != nil {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: err.Error()})
		}
		return
	}

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true})
	}

	go app.lockedRunComposeAction(stackName, "unpause", "unpause")
}

// lockedRunComposeAction acquires the per-stack lock and runs runComposeAction.
func (app *App) lockedRunComposeAction(stackName, action string, composeArgs ...string) {
	app.StackLocks.Lock(stackName)
	defer app.StackLocks.Unlock(stackName)
	app.runComposeAction(stackName, action, composeArgs...)
}

// lockedRunUnmanagedStackAction acquires the per-stack lock and runs runUnmanagedStackAction.
func (app *App) lockedRunUnmanagedStackAction(stackName, action string, composeArgs ...string) {
	app.StackLocks.Lock(stackName)
	defer app.StackLocks.Unlock(stackName)
	app.runUnmanagedStackAction(stackName, action, composeArgs...)
}

// runComposeAction runs a compose command in the background, streaming output
// to a PTY terminal that fans out to WebSocket clients.
// In mock mode, exec.Command resolves to the mock docker binary via PATH.
func (app *App) runComposeAction(stackName, action string, composeArgs ...string) {
	termName := "compose-" + stackName
	envArgs := compose.GlobalEnvArgs(app.StacksDir, stackName)
	displayParts := append(envArgs, composeArgs...)
	cmdDisplay := "$ docker compose " + strings.Join(displayParts, " ") + "\r\n"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

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
			errMsg := "\r\n[Error] " + err.Error() + "\r\n"
			term.Write([]byte(errMsg))
			slog.Error("compose action", "action", action, "stack", stackName, "err", err)
		}
	} else {
		term.Write([]byte("\r\n[Done]\r\n"))
	}

	// Schedule terminal cleanup after a grace period
	app.Terms.RemoveAfter(termName, 30*time.Second)
}

// runUnmanagedStackAction runs a compose command for an unmanaged stack (no
// compose file on disk) using "docker compose -p <project>". Docker Compose v2
// discovers containers by their project label, so start/stop/restart/down work
// without a compose file.
func (app *App) runUnmanagedStackAction(stackName, action string, composeArgs ...string) {
	termName := "compose-" + stackName
	cmdDisplay := fmt.Sprintf("$ docker compose -p %s %s\r\n", stackName, strings.Join(composeArgs, " "))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	term := app.Terms.Recreate(termName, terminal.TypePTY)
	term.Write([]byte(cmdDisplay))

	cmdArgs := []string{"compose", "-p", stackName}
	cmdArgs = append(cmdArgs, composeArgs...)
	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)

	if err := term.RunPTY(cmd); err != nil {
		if ctx.Err() == nil {
			errMsg := "\r\n[Error] " + err.Error() + "\r\n"
			term.Write([]byte(errMsg))
			slog.Error("unmanaged stack action", "action", action, "stack", stackName, "err", err)
		}
	} else {
		term.Write([]byte("\r\n[Done]\r\n"))
	}

	// Schedule terminal cleanup after a grace period
	app.Terms.RemoveAfter(termName, 30*time.Second)
}

// runDeployWithValidation validates the compose file via `docker compose config`
// and then runs `docker compose up -d --remove-orphans`.
func (app *App) runDeployWithValidation(stackName string) {
	termName := "compose-" + stackName
	envArgs := compose.GlobalEnvArgs(app.StacksDir, stackName)
	envDisplay := ""
	if len(envArgs) > 0 {
		envDisplay = strings.Join(envArgs, " ") + " "
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	term := app.Terms.Recreate(termName, terminal.TypePTY)
	dir := filepath.Join(app.StacksDir, stackName)

	// Step 1: Validate
	term.Write([]byte("$ docker compose " + envDisplay + "config --dry-run\r\n"))
	validateArgs := []string{"compose"}
	validateArgs = append(validateArgs, envArgs...)
	validateArgs = append(validateArgs, "config", "--dry-run")
	validateCmd := exec.CommandContext(ctx, "docker", validateArgs...)
	validateCmd.Dir = dir
	if err := term.RunPTY(validateCmd); err != nil {
		if ctx.Err() == nil {
			errMsg := "\r\n[Error] Validation failed: " + err.Error() + "\r\n"
			term.Write([]byte(errMsg))
			slog.Warn("deploy validation failed", "stack", stackName, "err", err)
		}
		// The compose file was already saved to disk — fsnotify detects the
		// new directory and triggers the stacks broadcast automatically.
		return
	}

	// Step 2: Deploy
	term.Write([]byte("$ docker compose " + envDisplay + "up -d --remove-orphans\r\n"))
	upArgs := []string{"compose"}
	upArgs = append(upArgs, envArgs...)
	upArgs = append(upArgs, "up", "-d", "--remove-orphans")
	upCmd := exec.CommandContext(ctx, "docker", upArgs...)
	upCmd.Dir = dir
	if err := term.RunPTY(upCmd); err != nil {
		if ctx.Err() == nil {
			errMsg := "\r\n[Error] " + err.Error() + "\r\n"
			term.Write([]byte(errMsg))
			slog.Error("compose action", "action", "deploy", "stack", stackName, "err", err)
		}
	} else {
		term.Write([]byte("\r\n[Done]\r\n"))
	}

	// Schedule terminal cleanup after a grace period
	app.Terms.RemoveAfter(termName, 30*time.Second)
}

// runDockerCommands runs multiple docker commands sequentially on the same terminal.
func (app *App) runDockerCommands(stackName, action string, argSets [][]string) {
	termName := "compose-" + stackName
	envArgs := compose.GlobalEnvArgs(app.StacksDir, stackName)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	term := app.Terms.Recreate(termName, terminal.TypePTY)
	dir := filepath.Join(app.StacksDir, stackName)

	for _, dockerArgs := range argSets {
		cmdDisplay := "$ docker " + strings.Join(composeEnvDisplay(dockerArgs, envArgs), " ") + "\r\n"
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
				errMsg := "\r\n[Error] " + err.Error() + "\r\n"
				term.Write([]byte(errMsg))
				slog.Error("compose action", "action", action, "stack", stackName, "err", err)
			}
			return
		}
	}

	term.Write([]byte("\r\n[Done]\r\n"))

	// Schedule terminal cleanup after a grace period
	app.Terms.RemoveAfter(termName, 30*time.Second)
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

