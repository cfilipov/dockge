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
			c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack and service name required"})
		}
		return
	}

	if msg.ID != nil {
		c.SendAck(*msg.ID, ws.OkResponse{OK: true, Msg: "Started"})
	}

	go app.runServiceAction(stackName, serviceName, "up", "up", "-d", serviceName)
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
			c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack and service name required"})
		}
		return
	}

	if msg.ID != nil {
		c.SendAck(*msg.ID, ws.OkResponse{OK: true, Msg: "Stopped"})
	}

	go app.runServiceAction(stackName, serviceName, "stop", "stop", serviceName)
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
			c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack and service name required"})
		}
		return
	}

	if msg.ID != nil {
		c.SendAck(*msg.ID, ws.OkResponse{OK: true, Msg: "Restarted"})
	}

	go app.runServiceAction(stackName, serviceName, "restart", "restart", serviceName)
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
			c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Stack and service name required"})
		}
		return
	}

	if msg.ID != nil {
		c.SendAck(*msg.ID, ws.OkResponse{OK: true, Msg: "Updated"})
	}

	go func() {
		app.runServiceAction(stackName, serviceName, "pull", "pull", serviceName)
		app.runServiceAction(stackName, serviceName, "up", "up", "-d", serviceName)
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
	termName := "compose--" + stackName
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

func (app *App) handleCheckImageUpdates(c *ws.Conn, msg *ws.ClientMessage) {
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

	go func() {
		app.checkImageUpdatesForStack(stackName)
		app.BroadcastAll()
	}()

	// Ack immediately — the check runs asynchronously
	if msg.ID != nil {
		c.SendAck(*msg.ID, map[string]interface{}{
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
func (app *App) StartImageUpdateChecker(ctx context.Context) {
	go func() {
		// Short delay on startup so the stack list loads first
		select {
		case <-ctx.Done():
			return
		case <-time.After(30 * time.Second):
		}

		if app.isImageUpdateCheckEnabled() {
			app.checkAllImageUpdates()
			app.BroadcastAll()
		}

		for {
			interval := app.getImageUpdateInterval()
			select {
			case <-ctx.Done():
				return
			case <-time.After(interval):
				if app.isImageUpdateCheckEnabled() {
					app.checkAllImageUpdates()
					app.BroadcastAll()
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
