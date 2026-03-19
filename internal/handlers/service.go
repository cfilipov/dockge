package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cfilipov/dockge/internal/compose"
	"github.com/cfilipov/dockge/internal/models"
	"github.com/cfilipov/dockge/internal/stack"
	"github.com/cfilipov/dockge/internal/terminal"
	"github.com/cfilipov/dockge/internal/ws"
)

const (
	defaultImageUpdateInterval = 6 * time.Hour
	imageCheckConcurrency      = 3
)

func RegisterServiceHandlers(app *App) {
	app.WS.Handle("startService", app.handleStartService)
	app.WS.Handle("stopService", app.handleStopService)
	app.WS.Handle("restartService", app.handleRestartService)
	app.WS.Handle("recreateService", app.handleRecreateService)
	app.WS.Handle("updateService", app.handleUpdateService)
	app.WS.Handle("checkImageUpdates", app.handleCheckImageUpdates)

	// Standalone container actions (no compose project label)
	app.WS.Handle("startContainer", app.handleStartContainer)
	app.WS.Handle("stopContainer", app.handleStopContainer)
	app.WS.Handle("restartContainer", app.handleRestartContainer)
}

func (app *App) handleStartService(c *ws.Conn, msg *ws.ClientMessage) {
	if checkLogin(c, msg) == 0 {
		return
	}
	args := parseArgs(msg)
	stackName := argString(args, 0)
	serviceName := argString(args, 1)
	if stackName == "" || serviceName == "" {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack and service name required"})
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
		go app.runServiceAction(stackName, serviceName, "up", "up", "-d", serviceName)
	} else {
		// Unmanaged: no compose file, use plain docker start on the container
		containerName := stackName + "-" + serviceName + "-1"
		go app.runContainerActionForStack(stackName, containerName, "start")
	}
}

func (app *App) handleStopService(c *ws.Conn, msg *ws.ClientMessage) {
	if checkLogin(c, msg) == 0 {
		return
	}
	args := parseArgs(msg)
	stackName := argString(args, 0)
	serviceName := argString(args, 1)
	if stackName == "" || serviceName == "" {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack and service name required"})
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
		go app.runServiceAction(stackName, serviceName, "stop", "stop", serviceName)
	} else {
		// Unmanaged: no compose file, use plain docker stop on the container
		containerName := stackName + "-" + serviceName + "-1"
		go app.runContainerActionForStack(stackName, containerName, "stop")
	}
}

func (app *App) handleRestartService(c *ws.Conn, msg *ws.ClientMessage) {
	if checkLogin(c, msg) == 0 {
		return
	}
	args := parseArgs(msg)
	stackName := argString(args, 0)
	serviceName := argString(args, 1)
	if stackName == "" || serviceName == "" {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack and service name required"})
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
		go app.runServiceAction(stackName, serviceName, "restart", "restart", serviceName)
	} else {
		// Unmanaged: no compose file, use plain docker restart on the container
		containerName := stackName + "-" + serviceName + "-1"
		go app.runContainerActionForStack(stackName, containerName, "restart")
	}
}

func (app *App) handleRecreateService(c *ws.Conn, msg *ws.ClientMessage) {
	if checkLogin(c, msg) == 0 {
		return
	}
	args := parseArgs(msg)
	stackName := argString(args, 0)
	serviceName := argString(args, 1)
	if stackName == "" || serviceName == "" {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack and service name required"})
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

	if !app.isStackManaged(stackName) {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Cannot recreate: stack is not managed by Dockge"})
		}
		return
	}

	go app.runServiceAction(stackName, serviceName, "recreate", "up", "-d", "--force-recreate", serviceName)
}

func (app *App) handleUpdateService(c *ws.Conn, msg *ws.ClientMessage) {
	if checkLogin(c, msg) == 0 {
		return
	}
	args := parseArgs(msg)
	stackName := argString(args, 0)
	serviceName := argString(args, 1)
	if stackName == "" || serviceName == "" {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack and service name required"})
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

	if !app.isStackManaged(stackName) {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Cannot update: stack is not managed by Dockge"})
		}
		return
	}

	go func() {
		app.runServiceAction(stackName, serviceName, "pull", "pull", serviceName)
		app.runServiceAction(stackName, serviceName, "up", "up", "-d", "--force-recreate", serviceName)
		// Clear stale "update available" cache and re-check with new images
		if err := app.ImageUpdates.DeleteForStack(stackName); err != nil {
			slog.Warn("clear image update cache", "stack", stackName, "err", err)
		}
		app.checkImageUpdatesForStack(stackName)
	}()
}

// runServiceAction runs a per-service compose command, streaming output to the
// stack's compose terminal (same terminal used by stack-level actions).
// In mock mode, exec.Command resolves to the mock docker binary via PATH.
func (app *App) runServiceAction(stackName, serviceName, action string, composeArgs ...string) {
	termName := "compose-" + stackName
	envArgs := compose.GlobalEnvArgs(app.StacksDir, stackName)
	displayParts := append(envArgs, composeArgs...)
	cmdDisplay := fmt.Sprintf("$ docker compose %s\r\n", strings.Join(displayParts, " "))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	term := app.Terms.Recreate(termName, terminal.TypePTY)
	term.Write([]byte(cmdDisplay))

	dir := filepath.Join(app.StacksDir, stackName)
	cmdArgs := []string{"compose"}
	cmdArgs = append(cmdArgs, envArgs...)
	cmdArgs = append(cmdArgs, composeArgs...)
	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	cmd.Dir = dir

	if err := term.RunPTY(cmd); err != nil {
		if ctx.Err() == nil {
			errMsg := fmt.Sprintf("\r\n[Error] %s\r\n", err.Error())
			term.Write([]byte(errMsg))
			slog.Error("service action", "action", action, "stack", stackName, "service", serviceName, "err", err)
		}
	} else {
		term.Write([]byte("\r\n[Done]\r\n"))
	}

	// Schedule terminal cleanup after a grace period
	app.Terms.RemoveAfter(termName, 30*time.Second)
}

// isStackManaged returns true if the stack has a compose file in the stacks directory.
func (app *App) isStackManaged(stackName string) bool {
	return compose.FindComposeFile(app.StacksDir, stackName) != ""
}

// runContainerActionForStack runs a plain docker command (start/stop/restart)
// for an unmanaged service container, writing output to the stack's compose
// terminal so the Compose page progress terminal shows the output.
func (app *App) runContainerActionForStack(stackName, containerName, action string) {
	termName := "compose-" + stackName
	cmdDisplay := fmt.Sprintf("$ docker %s %s\r\n", action, containerName)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	term := app.Terms.Recreate(termName, terminal.TypePTY)
	term.Write([]byte(cmdDisplay))

	cmd := exec.CommandContext(ctx, "docker", action, containerName)
	if err := term.RunPTY(cmd); err != nil {
		if ctx.Err() == nil {
			errMsg := fmt.Sprintf("\r\n[Error] %s\r\n", err.Error())
			term.Write([]byte(errMsg))
			slog.Error("unmanaged container action", "action", action, "stack", stackName, "container", containerName, "err", err)
		}
	} else {
		term.Write([]byte("\r\n[Done]\r\n"))
	}

	app.Terms.RemoveAfter(termName, 30*time.Second)
}

// --- Standalone container actions (no compose project) ---

func (app *App) handleStartContainer(c *ws.Conn, msg *ws.ClientMessage) {
	if checkLogin(c, msg) == 0 {
		return
	}
	args := parseArgs(msg)
	containerName := argString(args, 0)
	if containerName == "" {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Container name required"})
		}
		return
	}

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true})
	}

	go app.runContainerAction(containerName, "start")
}

func (app *App) handleStopContainer(c *ws.Conn, msg *ws.ClientMessage) {
	if checkLogin(c, msg) == 0 {
		return
	}
	args := parseArgs(msg)
	containerName := argString(args, 0)
	if containerName == "" {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Container name required"})
		}
		return
	}

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true})
	}

	go app.runContainerAction(containerName, "stop")
}

func (app *App) handleRestartContainer(c *ws.Conn, msg *ws.ClientMessage) {
	if checkLogin(c, msg) == 0 {
		return
	}
	args := parseArgs(msg)
	containerName := argString(args, 0)
	if containerName == "" {
		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "Container name required"})
		}
		return
	}

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true})
	}

	go app.runContainerAction(containerName, "restart")
}

// runContainerAction runs a plain docker command (start/stop/restart) for a
// standalone container that has no compose project association.
func (app *App) runContainerAction(containerName, action string) {
	termName := "container-" + containerName
	cmdDisplay := fmt.Sprintf("$ docker %s %s\r\n", action, containerName)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	term := app.Terms.Recreate(termName, terminal.TypePTY)
	term.Write([]byte(cmdDisplay))

	cmd := exec.CommandContext(ctx, "docker", action, containerName)
	if err := term.RunPTY(cmd); err != nil {
		if ctx.Err() == nil {
			errMsg := fmt.Sprintf("\r\n[Error] %s\r\n", err.Error())
			term.Write([]byte(errMsg))
			slog.Error("container action", "action", action, "container", containerName, "err", err)
		}
	} else {
		term.Write([]byte("\r\n[Done]\r\n"))
	}

	// Schedule terminal cleanup after a grace period
	app.Terms.RemoveAfter(termName, 30*time.Second)
}

func (app *App) handleCheckImageUpdates(c *ws.Conn, msg *ws.ClientMessage) {
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

	go func() {
		app.checkImageUpdatesForStack(stackName)
		app.TriggerUpdatesBroadcast()
	}()

	// Ack immediately — the check runs asynchronously
	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, map[string]interface{}{
			"ok":      true,
			"updated": true,
		})
	}
}

// Per-image timeout for digest lookups. Each image gets its own timeout
// so a slow/unreachable registry doesn't block checks for other images.
const perImageCheckTimeout = 30 * time.Second

// checkImageUpdatesForStack checks all services in a single stack for image updates.
// Reads compose data from disk (no cache). Respects dockge.imageupdates.check labels.
// Each image gets its own timeout so a slow registry doesn't block others.
func (app *App) checkImageUpdatesForStack(stackName string) {
	// Parse compose file from disk
	path := compose.FindComposeFile(app.StacksDir, stackName)
	if path == "" {
		return
	}
	serviceData := compose.ParseFile(path)
	if len(serviceData) == 0 {
		return
	}

	anyUpdate := false
	var failed int
	for svc, sd := range serviceData {
		if sd.Image == "" {
			continue
		}

		// Skip services with image update checking disabled
		if !sd.ImageUpdatesCheck {
			// Clear any stale BBolt entry
			if err := app.ImageUpdates.DeleteService(stackName, svc); err != nil {
				slog.Warn("delete disabled service update entry", "err", err, "stack", stackName, "svc", svc)
			}
			continue
		}

		imageRef := sd.Image

		// Per-image timeout — each image gets its own deadline
		imgCtx, imgCancel := context.WithTimeout(context.Background(), perImageCheckTimeout)
		localDigest := imageDigest(imgCtx, app, imageRef)
		remoteDigest := manifestDigest(imgCtx, app, imageRef)
		imgCancel()

		// Determine check status
		checkStatus := models.CheckStatusOK
		if localDigest == "" || remoteDigest == "" {
			checkStatus = models.CheckStatusFailed
			failed++
			slog.Debug("image update check failed",
				"stack", stackName, "svc", svc, "image", imageRef,
				"localDigest", localDigest != "", "remoteDigest", remoteDigest != "")
		}

		hasUpdate := checkStatus == models.CheckStatusOK && localDigest != remoteDigest
		if hasUpdate {
			anyUpdate = true
		}

		if err := app.ImageUpdates.Upsert(stackName, svc, imageRef, localDigest, remoteDigest, hasUpdate, checkStatus); err != nil {
			slog.Error("checkImageUpdates upsert", "err", err, "stack", stackName, "svc", svc)
		}
	}

	slog.Debug("image update check complete", "stack", stackName, "anyUpdate", anyUpdate, "failed", failed)
}

// imageDigest returns the local digest for an image using the Docker client.
func imageDigest(ctx context.Context, app *App, imageRef string) string {
	digests, err := app.Docker.ImageInspect(ctx, imageRef)
	if err != nil || len(digests) == 0 {
		return ""
	}
	// RepoDigests are in the form "repo@sha256:abc..."
	for _, d := range digests {
		if idx := strings.Index(d, "@"); idx >= 0 {
			return d[idx+1:]
		}
	}
	return digests[0]
}

// manifestDigest returns the remote (registry) digest for an image using the Docker client.
func manifestDigest(ctx context.Context, app *App, imageRef string) string {
	digest, err := app.Docker.DistributionInspect(ctx, imageRef)
	if err != nil {
		return ""
	}
	return digest
}

// getImageUpdateInterval reads the check interval from settings (in hours).
// Falls back to defaultImageUpdateInterval if not set or invalid.
func (app *App) getImageUpdateInterval() time.Duration {
	val, err := app.Settings.Get("imageUpdateCheckInterval")
	if err != nil || val == "" {
		return defaultImageUpdateInterval
	}
	hours, err := strconv.ParseFloat(val, 64)
	if err != nil || hours <= 0 {
		return defaultImageUpdateInterval
	}
	return time.Duration(hours * float64(time.Hour))
}

// isImageUpdateCheckEnabled reads the enabled flag from settings.
// Defaults to true if not set.
func (app *App) isImageUpdateCheckEnabled() bool {
	val, err := app.Settings.Get("imageUpdateCheckEnabled")
	if err != nil || val == "" {
		return true // enabled by default
	}
	return val != "0" && val != "false"
}

// StartImageUpdateChecker starts a background goroutine that periodically checks
// all stacks for image updates. Respects the imageUpdateCheckEnabled and
// imageUpdateCheckInterval settings, re-reading them on each tick so changes
// take effect without a restart.
//
// On startup, reads the last check timestamp from BoltDB. If the interval has
// elapsed (or no check has ever been done), runs immediately. This is
// mode-agnostic: against a fresh mock environment there's no stored timestamp
// so it checks right away; against a real daemon after a restart it skips if
// it checked recently.
func (app *App) StartImageUpdateChecker(ctx context.Context) {
	go func() {
		interval := app.getImageUpdateInterval()

		// Check if enough time has elapsed since the last check
		lastCheck, _ := app.ImageUpdates.GetLastCheckTime()
		elapsed := time.Since(lastCheck)
		if elapsed < interval {
			// Wait for the remaining time before the first check
			remaining := interval - elapsed
			slog.Debug("image update checker: deferring first check", "remaining", remaining)
			select {
			case <-ctx.Done():
				return
			case <-time.After(remaining):
			}
		}

		// Run the first check
		if app.isImageUpdateCheckEnabled() {
			app.checkAllImageUpdates()
			app.ImageUpdates.SetLastCheckTime(time.Now())
			app.TriggerUpdatesBroadcast()
		}

		for {
			interval = app.getImageUpdateInterval()
			select {
			case <-ctx.Done():
				return
			case <-time.After(interval):
				if app.isImageUpdateCheckEnabled() {
					app.checkAllImageUpdates()
					app.ImageUpdates.SetLastCheckTime(time.Now())
					app.TriggerUpdatesBroadcast()
				}
			}
		}
	}()
}

// checkAllImageUpdates iterates all stacks (from disk) and checks each for image updates,
// with a concurrency limit to avoid saturating the Docker daemon / network.
func (app *App) checkAllImageUpdates() {
	entries, err := os.ReadDir(app.StacksDir)
	if err != nil {
		slog.Warn("checkAllImageUpdates: read stacks dir", "err", err)
		return
	}

	// Collect stack names that have compose files
	var stackNames []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if compose.FindComposeFile(app.StacksDir, name) != "" {
			stackNames = append(stackNames, name)
		}
	}

	if len(stackNames) == 0 {
		return
	}

	slog.Info("background image update check starting", "stacks", len(stackNames))

	sem := make(chan struct{}, imageCheckConcurrency)
	var wg sync.WaitGroup

	for _, name := range stackNames {
		wg.Add(1)
		sem <- struct{}{}
		go func(stackName string) {
			defer wg.Done()
			defer func() { <-sem }()
			app.checkImageUpdatesForStack(stackName)
		}(name)
	}

	wg.Wait()
	slog.Debug("background image update check complete")
}
