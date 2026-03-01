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

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true, Msg: "Started"})
	}

	if app.isStackManaged(stackName) {
		go app.runServiceAction(stackName, serviceName, "up", "up", "-d", serviceName)
	} else {
		go app.runContainerAction(stackName, serviceName, "start")
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

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true, Msg: "Stopped"})
	}

	if app.isStackManaged(stackName) {
		go app.runServiceAction(stackName, serviceName, "stop", "stop", serviceName)
	} else {
		go app.runContainerAction(stackName, serviceName, "stop")
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

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true, Msg: "Restarted"})
	}

	if app.isStackManaged(stackName) {
		go app.runServiceAction(stackName, serviceName, "restart", "restart", serviceName)
	} else {
		go app.runContainerAction(stackName, serviceName, "restart")
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

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true, Msg: "Recreated"})
	}

	if app.isStackManaged(stackName) {
		go app.runServiceAction(stackName, serviceName, "recreate", "up", "-d", "--force-recreate", serviceName)
	} else {
		// No compose file — restart is the closest equivalent
		go app.runContainerAction(stackName, serviceName, "restart")
	}
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

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true, Msg: "Updated"})
	}

	if app.isStackManaged(stackName) {
		go func() {
			app.runServiceAction(stackName, serviceName, "pull", "pull", serviceName)
			app.runServiceAction(stackName, serviceName, "up", "up", "-d", "--force-recreate", serviceName)
			// Clear stale "update available" cache and re-check with new images
			if err := app.ImageUpdates.DeleteForStack(stackName); err != nil {
				slog.Warn("clear image update cache", "stack", stackName, "err", err)
			}
			app.checkImageUpdatesForStack(stackName)
		}()
	} else {
		go app.runContainerPullAndRestart(stackName, serviceName)
	}
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
}

// isStackManaged returns true if the stack has a compose file in the stacks directory.
func (app *App) isStackManaged(stackName string) bool {
	return compose.FindComposeFile(app.StacksDir, stackName) != ""
}

// findContainerName looks up the actual container name from Docker by project+service labels.
func (app *App) findContainerName(stackName, serviceName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	containers, err := app.Docker.ContainerList(ctx, true, stackName)
	if err != nil {
		return "", fmt.Errorf("container list: %w", err)
	}
	for _, c := range containers {
		if c.Service == serviceName {
			return c.Name, nil
		}
	}
	return "", fmt.Errorf("container not found for %s/%s", stackName, serviceName)
}

// runContainerAction runs a plain docker command (stop/start/restart) for an
// unmanaged container, streaming output to the stack's terminal.
func (app *App) runContainerAction(stackName, serviceName, action string) {
	termName := "compose-" + stackName

	containerName, err := app.findContainerName(stackName, serviceName)
	if err != nil {
		term := app.Terms.Recreate(termName, terminal.TypePTY)
		errMsg := fmt.Sprintf("[Error] %s\r\n", err.Error())
		term.Write([]byte(errMsg))
		slog.Error("container action", "action", action, "stack", stackName, "service", serviceName, "err", err)
		return
	}

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
			slog.Error("container action", "action", action, "stack", stackName, "service", serviceName, "err", err)
		}
	} else {
		term.Write([]byte("\r\n[Done]\r\n"))
	}
}

// runContainerPullAndRestart pulls the latest image for an unmanaged container
// and restarts it. This is the unmanaged equivalent of "update".
func (app *App) runContainerPullAndRestart(stackName, serviceName string) {
	termName := "compose-" + stackName

	containerName, err := app.findContainerName(stackName, serviceName)
	if err != nil {
		term := app.Terms.Recreate(termName, terminal.TypePTY)
		errMsg := fmt.Sprintf("[Error] %s\r\n", err.Error())
		term.Write([]byte(errMsg))
		return
	}

	// Get the image name from the container's inspect data
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	containers, err := app.Docker.ContainerList(ctx, true, stackName)
	if err != nil {
		term := app.Terms.Recreate(termName, terminal.TypePTY)
		errMsg := fmt.Sprintf("[Error] %s\r\n", err.Error())
		term.Write([]byte(errMsg))
		return
	}
	var imageName string
	for _, c := range containers {
		if c.Service == serviceName {
			imageName = c.Image
			break
		}
	}
	if imageName == "" {
		term := app.Terms.Recreate(termName, terminal.TypePTY)
		term.Write([]byte("[Error] could not determine image for container\r\n"))
		return
	}

	// Pull the image
	term := app.Terms.Recreate(termName, terminal.TypePTY)
	pullDisplay := fmt.Sprintf("$ docker pull %s\r\n", imageName)
	term.Write([]byte(pullDisplay))

	pullCmd := exec.CommandContext(ctx, "docker", "pull", imageName)
	if err := term.RunPTY(pullCmd); err != nil {
		if ctx.Err() == nil {
			errMsg := fmt.Sprintf("\r\n[Error] %s\r\n", err.Error())
			term.Write([]byte(errMsg))
		}
		return
	}
	term.Write([]byte("\r\n[Done]\r\n"))

	// Restart the container
	restartDisplay := fmt.Sprintf("$ docker restart %s\r\n", containerName)
	term.Write([]byte(restartDisplay))

	restartCmd := exec.CommandContext(ctx, "docker", "restart", containerName)
	if err := term.RunPTY(restartCmd); err != nil {
		if ctx.Err() == nil {
			errMsg := fmt.Sprintf("\r\n[Error] %s\r\n", err.Error())
			term.Write([]byte(errMsg))
		}
	} else {
		term.Write([]byte("\r\n[Done]\r\n"))
	}
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

// checkImageUpdatesForStack checks all services in a single stack for image updates.
// Reads compose data from disk (no cache). Respects dockge.imageupdates.check labels.
func (app *App) checkImageUpdatesForStack(stackName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

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
		localDigest := imageDigest(ctx, app, imageRef)
		remoteDigest := manifestDigest(ctx, app, imageRef)

		hasUpdate := localDigest != "" && remoteDigest != "" && localDigest != remoteDigest
		if hasUpdate {
			anyUpdate = true
		}

		if err := app.ImageUpdates.Upsert(stackName, svc, imageRef, localDigest, remoteDigest, hasUpdate); err != nil {
			slog.Error("checkImageUpdates upsert", "err", err, "stack", stackName, "svc", svc)
		}
	}

	slog.Debug("image update check complete", "stack", stackName, "anyUpdate", anyUpdate)
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
		// Short delay on startup so the stack list loads first
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}

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
