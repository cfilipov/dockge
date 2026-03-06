package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cfilipov/dockge/internal/compose"
	"github.com/cfilipov/dockge/internal/docker"
	"github.com/cfilipov/dockge/internal/ws"
)

// Broadcast channel names.
const (
	chanStacks     = "stacks"
	chanContainers = "containers"
	chanNetworks   = "networks"
	chanImages     = "images"
	chanVolumes    = "volumes"
	chanUpdates    = "updates"
)

// StackBroadcastEntry is the per-stack data sent via the stacks broadcast channel.
// Status is NOT included — the frontend derives it from the container store.
type StackBroadcastEntry struct {
	Name            string                       `json:"name"`
	ComposeFileName string                       `json:"composeFileName"`
	IgnoreStatus    map[string]bool              `json:"ignoreStatus,omitempty"`
	Images          map[string]string            `json:"images"`
	IsManagedByDockge bool                       `json:"isManagedByDockge"`
}

// dispatchWork is sent through the dispatch channel to the worker goroutine.
type dispatchWork struct {
	evt      docker.DockerEvent
	fullSync string // non-empty = full refresh for this channel (bypass event routing)
}

// BroadcastMetrics tracks per-channel broadcast statistics.
type BroadcastMetrics struct {
	mu       sync.Mutex
	counters map[string]*ChannelMetrics
}

// ChannelMetrics holds counters for a single broadcast channel.
type ChannelMetrics struct {
	Triggered int64 `json:"triggered"` // events received
	Sent      int64 `json:"sent"`      // broadcasts sent
}

func newBroadcastMetrics() *BroadcastMetrics {
	channels := []string{chanStacks, chanContainers, chanNetworks, chanImages, chanVolumes, chanUpdates}
	m := &BroadcastMetrics{
		counters: make(map[string]*ChannelMetrics, len(channels)),
	}
	for _, ch := range channels {
		m.counters[ch] = &ChannelMetrics{}
	}
	return m
}

func (bm *BroadcastMetrics) recordTriggered(channel string) {
	bm.mu.Lock()
	if cm, ok := bm.counters[channel]; ok {
		cm.Triggered++
	}
	bm.mu.Unlock()
}

func (bm *BroadcastMetrics) recordSent(channel string) {
	bm.mu.Lock()
	if cm, ok := bm.counters[channel]; ok {
		cm.Sent++
	}
	bm.mu.Unlock()
}

// Snapshot returns a copy of all metrics for JSON serialization.
func (bm *BroadcastMetrics) Snapshot() map[string]*ChannelMetrics {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	result := make(map[string]*ChannelMetrics, len(bm.counters))
	for k, v := range bm.counters {
		result[k] = &ChannelMetrics{
			Triggered: v.Triggered,
			Sent:      v.Sent,
		}
	}
	return result
}

// sendToConn sends channel data to a single connection (used for initial connect).
func sendToConn(c *ws.Conn, channel string, data any) {
	ws.SendEvent(c, channel, data)
}

// --- Map-building helpers ---

// mapPayload wraps a map with a replace flag. When Replace is true, the
// frontend clears the store before merging. Used for full-list broadcasts
// (initial load, Trigger*). Event-driven partial updates send bare maps.
type mapPayload struct {
	Replace bool           `json:"replace"`
	Data    map[string]any `json:"data"`
}

// containersToMap converts a slice of ContainerBroadcast to a map keyed by name.
func containersToMap(containers []docker.ContainerBroadcast) map[string]any {
	m := make(map[string]any, len(containers))
	for _, c := range containers {
		m[c.Name] = c
	}
	return m
}

// networksToMap converts a slice of NetworkSummary to a map keyed by name.
func networksToMap(networks []docker.NetworkSummary) map[string]any {
	m := make(map[string]any, len(networks))
	for _, n := range networks {
		m[n.Name] = n
	}
	return m
}

// imagesToMap converts a slice of ImageSummary to a map keyed by ID.
func imagesToMap(images []docker.ImageSummary) map[string]any {
	m := make(map[string]any, len(images))
	for _, img := range images {
		m[img.ID] = img
	}
	return m
}

// volumesToMap converts a slice of VolumeSummary to a map keyed by name.
func volumesToMap(volumes []docker.VolumeSummary) map[string]any {
	m := make(map[string]any, len(volumes))
	for _, v := range volumes {
		m[v.Name] = v
	}
	return m
}

// stacksToMap converts a slice of StackBroadcastEntry to a map keyed by name.
func stacksToMap(stacks []StackBroadcastEntry) map[string]any {
	m := make(map[string]any, len(stacks))
	for _, s := range stacks {
		m[s.Name] = s
	}
	return m
}

// --- Full-list broadcast functions (for initial load + Trigger methods) ---

// broadcastStacksMap queries stacks and broadcasts as a full-replace map.
func (app *App) broadcastStacksMap() {
	if !app.WS.HasAuthenticatedConns() {
		return
	}
	entries := buildStackBroadcast(app.StacksDir)
	ws.BroadcastAuthenticated(app.WS, chanStacks, mapPayload{Replace: true, Data: stacksToMap(entries)})
	app.BcastMetrics.recordSent(chanStacks)
}

// broadcastContainersMap queries Docker for all containers and broadcasts as a full-replace map.
func (app *App) broadcastContainersMap() {
	if !app.WS.HasAuthenticatedConns() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	containers, err := app.Docker.ContainerListDetailed(ctx)
	if err != nil {
		slog.Warn("broadcastContainersMap", "err", err)
		containers = []docker.ContainerBroadcast{}
	}
	ws.BroadcastAuthenticated(app.WS, chanContainers, mapPayload{Replace: true, Data: containersToMap(containers)})
	app.BcastMetrics.recordSent(chanContainers)
}

// broadcastNetworksMap queries Docker for all networks and broadcasts as a full-replace map.
func (app *App) broadcastNetworksMap() {
	if !app.WS.HasAuthenticatedConns() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	networks, err := app.Docker.NetworkList(ctx)
	if err != nil {
		slog.Warn("broadcastNetworksMap", "err", err)
		networks = []docker.NetworkSummary{}
	}
	ws.BroadcastAuthenticated(app.WS, chanNetworks, mapPayload{Replace: true, Data: networksToMap(networks)})
	app.BcastMetrics.recordSent(chanNetworks)
}

// broadcastImagesMap queries Docker for all images and broadcasts as a full-replace map.
func (app *App) broadcastImagesMap() {
	if !app.WS.HasAuthenticatedConns() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	images, err := app.Docker.ImageList(ctx)
	if err != nil {
		slog.Warn("broadcastImagesMap", "err", err)
		images = []docker.ImageSummary{}
	}
	ws.BroadcastAuthenticated(app.WS, chanImages, mapPayload{Replace: true, Data: imagesToMap(images)})
	app.BcastMetrics.recordSent(chanImages)
}

// broadcastVolumesMap queries Docker for all volumes and broadcasts as a full-replace map.
func (app *App) broadcastVolumesMap() {
	if !app.WS.HasAuthenticatedConns() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	volumes, err := app.Docker.VolumeList(ctx)
	if err != nil {
		slog.Warn("broadcastVolumesMap", "err", err)
		volumes = []docker.VolumeSummary{}
	}
	ws.BroadcastAuthenticated(app.WS, chanVolumes, mapPayload{Replace: true, Data: volumesToMap(volumes)})
	app.BcastMetrics.recordSent(chanVolumes)
}

// broadcastUpdates reads BoltDB image update cache and broadcasts container names with updates.
func (app *App) broadcastUpdates() {
	svcUpdates, err := app.ImageUpdates.AllServiceUpdates()
	if err != nil {
		slog.Warn("broadcastUpdates", "err", err)
		svcUpdates = map[string]bool{}
	}

	updated := make([]string, 0, len(svcUpdates))
	for key, hasUpdate := range svcUpdates {
		if hasUpdate {
			updated = append(updated, key)
		}
	}
	sort.Strings(updated)

	ws.BroadcastAuthenticated(app.WS, chanUpdates, updated)
	app.BcastMetrics.recordSent(chanUpdates)
}

// buildStackBroadcast scans the stacks directory and builds the broadcast payload.
func buildStackBroadcast(stacksDir string) []StackBroadcastEntry {
	entries, err := os.ReadDir(stacksDir)
	if err != nil {
		slog.Warn("buildStackBroadcast: readdir", "err", err)
		return []StackBroadcastEntry{}
	}

	result := make([]StackBroadcastEntry, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		composeFile := compose.FindComposeFile(stacksDir, name)
		if composeFile == "" {
			continue
		}

		services := compose.ParseFile(composeFile)
		images := make(map[string]string, len(services))
		var ignoreStatus map[string]bool
		for svc, sd := range services {
			if sd.Image != "" {
				images[svc] = sd.Image
			}
			if sd.StatusIgnore {
				if ignoreStatus == nil {
					ignoreStatus = make(map[string]bool)
				}
				ignoreStatus[svc] = true
			}
		}

		result = append(result, StackBroadcastEntry{
			Name:              name,
			ComposeFileName:   filepath.Base(composeFile),
			IgnoreStatus:      ignoreStatus,
			Images:            images,
			IsManagedByDockge: true,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// --- Dispatch channel + worker (1+1 goroutine model) ---

// InitBroadcast initializes the broadcast state, dispatch channel, and metrics.
func (app *App) InitBroadcast() {
	app.dispatchCh = make(chan dispatchWork, 64)
	app.BcastMetrics = newBroadcastMetrics()
	app.EventBus = NewEventBus()
}

// StartBroadcastWatcher starts the event-driven broadcast system.
// It starts the dispatch worker and event consumer goroutines.
func (app *App) StartBroadcastWatcher(ctx context.Context) {
	slog.Info("broadcast watcher started")
	go app.runDispatchWorker(ctx)
	go app.runBroadcastWatcherLoop(ctx)
}

// runDispatchWorker processes dispatch work sequentially, preserving event ordering.
func (app *App) runDispatchWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case work := <-app.dispatchCh:
			if work.fullSync != "" {
				app.dispatchFullSync(ctx, work.fullSync)
			} else {
				app.dispatchEvent(ctx, work.evt)
			}
		}
	}
}

// dispatchFullSync handles a full-refresh broadcast for a channel.
func (app *App) dispatchFullSync(_ context.Context, channel string) {
	switch channel {
	case chanStacks:
		app.broadcastStacksMap()
	case chanContainers:
		app.broadcastContainersMap()
	case chanNetworks:
		app.broadcastNetworksMap()
	case chanImages:
		app.broadcastImagesMap()
	case chanVolumes:
		app.broadcastVolumesMap()
	case chanUpdates:
		app.broadcastUpdates()
	}
}

// dispatchEvent handles a single Docker event — queries only the affected
// resource and broadcasts a partial map update.
func (app *App) dispatchEvent(ctx context.Context, evt docker.DockerEvent) {
	if !app.WS.HasAuthenticatedConns() {
		return
	}

	dctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	switch evt.Type {
	case "container":
		app.dispatchContainerEvent(dctx, evt)
	case "network":
		app.dispatchNetworkEvent(dctx, evt)
	case "image":
		app.dispatchImageEvent(dctx, evt)
	case "volume":
		app.dispatchVolumeEvent(dctx, evt)
	}
}

// dispatchContainerEvent handles container lifecycle events.
func (app *App) dispatchContainerEvent(ctx context.Context, evt docker.DockerEvent) {
	app.BcastMetrics.recordTriggered(chanContainers)

	action := evt.Action
	// Strip health_status prefix to just "health_status"
	if strings.HasPrefix(action, "health_status") {
		action = "health_status"
	}

	switch action {
	case "destroy":
		// Container is gone — broadcast deletion by name
		name := evt.Name
		if name == "" {
			slog.Warn("container destroy event missing name", "id", evt.ContainerID)
			return
		}
		data := map[string]any{name: nil}
		ws.BroadcastAuthenticated(app.WS, chanContainers, data)
		app.BcastMetrics.recordSent(chanContainers)

	default:
		// Query the specific container
		containers, err := app.Docker.ContainerListDetailedByID(ctx, evt.ContainerID)
		if err != nil {
			slog.Warn("dispatch container", "err", err, "id", evt.ContainerID)
			return
		}
		if len(containers) == 0 {
			// Container might have already been removed
			return
		}
		ws.BroadcastAuthenticated(app.WS, chanContainers, containersToMap(containers))
		app.BcastMetrics.recordSent(chanContainers)
	}
}

// dispatchNetworkEvent handles network lifecycle events.
func (app *App) dispatchNetworkEvent(ctx context.Context, evt docker.DockerEvent) {
	switch evt.Action {
	case "connect", "disconnect":
		// Network connect/disconnect: the network metadata doesn't change,
		// but the container's network list does. Broadcast containers update.
		app.BcastMetrics.recordTriggered(chanContainers)
		if evt.ContainerID == "" {
			return
		}
		containers, err := app.Docker.ContainerListDetailedByID(ctx, evt.ContainerID)
		if err != nil {
			slog.Warn("dispatch network connect/disconnect", "err", err)
			return
		}
		if len(containers) > 0 {
			ws.BroadcastAuthenticated(app.WS, chanContainers, containersToMap(containers))
			app.BcastMetrics.recordSent(chanContainers)
		}

	case "destroy":
		// Network removed — broadcast deletion by name
		app.BcastMetrics.recordTriggered(chanNetworks)
		name := evt.Name
		if name == "" {
			slog.Warn("network destroy event missing name", "id", evt.ActorID)
			return
		}
		data := map[string]any{name: nil}
		ws.BroadcastAuthenticated(app.WS, chanNetworks, data)
		app.BcastMetrics.recordSent(chanNetworks)

	default:
		// Network created or other — query the specific network
		app.BcastMetrics.recordTriggered(chanNetworks)
		networks, err := app.Docker.NetworkListByID(ctx, evt.ActorID)
		if err != nil {
			slog.Warn("dispatch network", "err", err, "id", evt.ActorID)
			return
		}
		if len(networks) > 0 {
			ws.BroadcastAuthenticated(app.WS, chanNetworks, networksToMap(networks))
			app.BcastMetrics.recordSent(chanNetworks)
		}
	}
}

// dispatchImageEvent handles image lifecycle events.
func (app *App) dispatchImageEvent(_ context.Context, evt docker.DockerEvent) {
	app.BcastMetrics.recordTriggered(chanImages)

	switch evt.Action {
	case "delete":
		// Image removed — broadcast deletion by ID
		id := evt.ActorID
		if id == "" {
			return
		}
		data := map[string]any{id: nil}
		ws.BroadcastAuthenticated(app.WS, chanImages, data)
		app.BcastMetrics.recordSent(chanImages)

	default:
		// pull, tag, untag, etc. — do a full image list since we can't
		// reliably filter by the event's actor ID (it may be a tag name)
		app.broadcastImagesMap()
	}
}

// dispatchVolumeEvent handles volume lifecycle events.
func (app *App) dispatchVolumeEvent(ctx context.Context, evt docker.DockerEvent) {
	switch evt.Action {
	case "mount", "unmount":
		// Volume mount/unmount: volume metadata doesn't change,
		// but the container's mount list does.
		app.BcastMetrics.recordTriggered(chanContainers)
		if evt.ContainerID == "" {
			// Volume events carry the actor ID as the volume name, not container ID.
			// For mount/unmount we'd need the container — fall back to full refresh.
			app.broadcastContainersMap()
			return
		}
		containers, err := app.Docker.ContainerListDetailedByID(ctx, evt.ContainerID)
		if err != nil {
			slog.Warn("dispatch volume mount/unmount", "err", err)
			return
		}
		if len(containers) > 0 {
			ws.BroadcastAuthenticated(app.WS, chanContainers, containersToMap(containers))
			app.BcastMetrics.recordSent(chanContainers)
		}

	case "destroy":
		// Volume removed — broadcast deletion by name
		app.BcastMetrics.recordTriggered(chanVolumes)
		name := evt.Name
		if name == "" {
			name = evt.ActorID // volume events use actor ID as name
		}
		if name == "" {
			return
		}
		data := map[string]any{name: nil}
		ws.BroadcastAuthenticated(app.WS, chanVolumes, data)
		app.BcastMetrics.recordSent(chanVolumes)

	default:
		// Volume created — query the specific volume
		app.BcastMetrics.recordTriggered(chanVolumes)
		name := evt.Name
		if name == "" {
			name = evt.ActorID
		}
		if name == "" {
			return
		}
		volumes, err := app.Docker.VolumeListByName(ctx, name)
		if err != nil {
			slog.Warn("dispatch volume", "err", err, "name", name)
			return
		}
		if len(volumes) > 0 {
			ws.BroadcastAuthenticated(app.WS, chanVolumes, volumesToMap(volumes))
			app.BcastMetrics.recordSent(chanVolumes)
		}
	}
}

// runBroadcastWatcherLoop subscribes to Docker events and sends them to the
// dispatch channel. On error it retries with exponential backoff.
func (app *App) runBroadcastWatcherLoop(ctx context.Context) {
	const maxRetries = 5
	failures := 0
	backoff := 1 * time.Second

	for {
		eventCh, errCh := app.Docker.Events(ctx)

		err := app.consumeBroadcastEvents(ctx, eventCh, errCh)
		if ctx.Err() != nil {
			return // clean shutdown
		}

		failures++
		if failures > maxRetries {
			slog.Error("docker events (broadcast): too many failures, exiting", "failures", failures, "lastErr", err)
			os.Exit(1)
		}

		slog.Warn("docker events (broadcast): retrying", "attempt", failures, "backoff", backoff, "err", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, 30*time.Second)
	}
}

// consumeBroadcastEvents reads Docker events and sends them to the dispatch channel.
func (app *App) consumeBroadcastEvents(ctx context.Context, eventCh <-chan docker.DockerEvent, errCh <-chan error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case evt, ok := <-eventCh:
			if !ok {
				return fmt.Errorf("docker events channel closed")
			}
			slog.Debug("docker event", "type", evt.Type, "action", evt.Action, "name", evt.Name)

			// Fan out to per-terminal subscribers
			app.EventBus.Publish(evt)

			if !app.WS.HasAuthenticatedConns() {
				continue
			}

			// Broadcast raw Docker event for dev inspection
			if evt.Raw != nil {
				ws.BroadcastAuthenticated(app.WS, "dockerEvent", evt.Raw)
			} else {
				ws.BroadcastAuthenticated(app.WS, "dockerEvent", evt)
			}

			// Send to dispatch channel for granular processing
			select {
			case app.dispatchCh <- dispatchWork{evt: evt}:
			default:
				slog.Warn("dispatch channel full, dropping event", "type", evt.Type, "action", evt.Action)
			}

		case err, ok := <-errCh:
			if !ok {
				continue
			}
			return fmt.Errorf("docker events error: %w", err)
		}
	}
}

// --- Trigger methods (for explicit full refreshes) ---

// TriggerStacksBroadcast triggers a full stacks broadcast via dispatch channel.
func (app *App) TriggerStacksBroadcast() {
	select {
	case app.dispatchCh <- dispatchWork{fullSync: chanStacks}:
	default:
	}
}

// TriggerContainersBroadcast triggers a full containers broadcast via dispatch channel.
func (app *App) TriggerContainersBroadcast() {
	select {
	case app.dispatchCh <- dispatchWork{fullSync: chanContainers}:
	default:
	}
}

// TriggerNetworksBroadcast triggers a full networks broadcast via dispatch channel.
func (app *App) TriggerNetworksBroadcast() {
	select {
	case app.dispatchCh <- dispatchWork{fullSync: chanNetworks}:
	default:
	}
}

// TriggerImagesBroadcast triggers a full images broadcast via dispatch channel.
func (app *App) TriggerImagesBroadcast() {
	select {
	case app.dispatchCh <- dispatchWork{fullSync: chanImages}:
	default:
	}
}

// TriggerVolumesBroadcast triggers a full volumes broadcast via dispatch channel.
func (app *App) TriggerVolumesBroadcast() {
	select {
	case app.dispatchCh <- dispatchWork{fullSync: chanVolumes}:
	default:
	}
}

// TriggerUpdatesBroadcast triggers a full updates broadcast via dispatch channel.
func (app *App) TriggerUpdatesBroadcast() {
	select {
	case app.dispatchCh <- dispatchWork{fullSync: chanUpdates}:
	default:
	}
}
