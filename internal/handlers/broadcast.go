package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/cfilipov/dockge/internal/compose"
	"github.com/cfilipov/dockge/internal/docker"
	"github.com/cfilipov/dockge/internal/ws"
)

// Broadcast channel names.
const (
	chanStacks        = "stacks"
	chanContainers    = "containers"
	chanNetworks      = "networks"
	chanImages        = "images"
	chanVolumes       = "volumes"
	chanUpdates       = "updates"
	chanResourceEvent = "resourceEvent"
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

// ChannelBroadcast wraps broadcast data for resource channels.
// All resource channels (containers, networks, images, volumes, stacks) use this shape.
// Events are sent separately on the "resourceEvent" channel.
type ChannelBroadcast struct {
	Items map[string]any `json:"items"`
}

// ResourceEvent describes a Docker event that triggered a broadcast.
type ResourceEvent struct {
	Type        string `json:"type"`                  // "container", "network", "image", "volume"
	Action      string `json:"action"`                // start, stop, die, destroy, create, connect, ...
	ID          string `json:"id"`                    // Actor ID
	Name        string `json:"name"`                  // Resource name
	StackName   string `json:"stackName,omitempty"`   // com.docker.compose.project
	ServiceName string `json:"serviceName,omitempty"` // com.docker.compose.service
	ContainerID string `json:"containerId,omitempty"` // For network connect/disconnect
}

// toResourceEvent converts a DockerEvent to a ResourceEvent for broadcast.
func toResourceEvent(evt docker.DockerEvent) ResourceEvent {
	return ResourceEvent{
		Type:        evt.Type,
		Action:      evt.Action,
		ID:          evt.ActorID,
		Name:        evt.Name,
		StackName:   evt.Project,
		ServiceName: evt.Service,
		ContainerID: evt.ContainerID,
	}
}

// broadcastChannel sends a ChannelBroadcast on the given channel.
func (app *App) broadcastChannel(channel string, items map[string]any) {
	ws.BroadcastAuthenticated(app.WS, channel, ChannelBroadcast{
		Items: items,
	})
	app.BcastMetrics.recordSent(channel)
}

// sendToConn sends channel data to a single connection (used for initial connect).
// For resource channels, wraps data in ChannelBroadcast format.
func sendToConn(c *ws.Conn, channel string, data any) {
	switch channel {
	case chanStacks, chanContainers, chanNetworks, chanImages, chanVolumes:
		if m, ok := data.(map[string]any); ok {
			ws.SendEvent(c, channel, ChannelBroadcast{Items: m})
			return
		}
	}
	ws.SendEvent(c, channel, data)
}

// --- Map-building helpers ---

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
	app.broadcastChannel(chanStacks, stacksToMap(entries))
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
	app.broadcastChannel(chanContainers, containersToMap(containers))
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
	app.broadcastChannel(chanNetworks, networksToMap(networks))
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
	app.broadcastChannel(chanImages, imagesToMap(images))
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
	app.broadcastChannel(chanVolumes, volumesToMap(volumes))
}

// broadcastContainersByIDs queries Docker for specific containers using batched
// list call and broadcasts a partial map. Falls back to full list if >25 IDs.
func (app *App) broadcastContainersByIDs(ids map[string]bool, destroyed []string) {
	if !app.WS.HasAuthenticatedConns() {
		return
	}
	if len(ids) > 25 {
		app.broadcastContainersMap()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	idSlice := make([]string, 0, len(ids))
	for id := range ids {
		idSlice = append(idSlice, id)
	}
	containers, err := app.Docker.ContainerListDetailedByIDs(ctx, idSlice)
	if err != nil {
		slog.Warn("broadcastContainersByIDs", "err", err)
		return
	}
	m := containersToMap(containers)
	for _, name := range destroyed {
		if _, exists := m[name]; !exists {
			m[name] = nil
		}
	}
	if len(m) > 0 {
		app.broadcastChannel(chanContainers, m)
	}
}

// broadcastNetworksByIDs queries Docker for specific networks using batched
// list call and broadcasts a partial map. Falls back to full list if >25 IDs.
func (app *App) broadcastNetworksByIDs(ids map[string]bool, destroyed []string) {
	if !app.WS.HasAuthenticatedConns() {
		return
	}
	if len(ids) > 25 {
		app.broadcastNetworksMap()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	idSlice := make([]string, 0, len(ids))
	for id := range ids {
		idSlice = append(idSlice, id)
	}
	networks, err := app.Docker.NetworkListByIDs(ctx, idSlice)
	if err != nil {
		slog.Warn("broadcastNetworksByIDs", "err", err)
		return
	}
	m := networksToMap(networks)
	for _, name := range destroyed {
		if _, exists := m[name]; !exists {
			m[name] = nil
		}
	}
	if len(m) > 0 {
		app.broadcastChannel(chanNetworks, m)
	}
}

// broadcastImagesByIDs queries Docker for specific images using batched
// list call and broadcasts a partial map. Falls back to full list if >25 IDs.
func (app *App) broadcastImagesByIDs(ids map[string]bool, destroyed []string) {
	if !app.WS.HasAuthenticatedConns() {
		return
	}
	if len(ids) > 25 {
		app.broadcastImagesMap()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	idSlice := make([]string, 0, len(ids))
	for id := range ids {
		idSlice = append(idSlice, id)
	}
	images, err := app.Docker.ImageListByIDs(ctx, idSlice)
	if err != nil {
		slog.Warn("broadcastImagesByIDs", "err", err)
		return
	}
	m := imagesToMap(images)
	for _, id := range destroyed {
		if _, exists := m[id]; !exists {
			m[id] = nil
		}
	}
	if len(m) > 0 {
		app.broadcastChannel(chanImages, m)
	}
}

// broadcastVolumesByNames queries Docker for specific volumes using batched
// list call and broadcasts a partial map. Falls back to full list if >25 names.
func (app *App) broadcastVolumesByNames(names map[string]bool, destroyed []string) {
	if !app.WS.HasAuthenticatedConns() {
		return
	}
	if len(names) > 25 {
		app.broadcastVolumesMap()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	nameSlice := make([]string, 0, len(names))
	for name := range names {
		nameSlice = append(nameSlice, name)
	}
	volumes, err := app.Docker.VolumeListByNames(ctx, nameSlice)
	if err != nil {
		slog.Warn("broadcastVolumesByNames", "err", err)
		return
	}
	m := volumesToMap(volumes)
	for _, name := range destroyed {
		if _, exists := m[name]; !exists {
			m[name] = nil
		}
	}
	if len(m) > 0 {
		app.broadcastChannel(chanVolumes, m)
	}
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

// Coalescing parameters for the dispatch worker.
const (
	// dispatchQuietPeriod is how long to wait for more events before
	// processing a batch. Resets on each new event.
	dispatchQuietPeriod = 50 * time.Millisecond

	// dispatchMaxBatch is the maximum time to collect events before
	// forcing a flush, even if events keep arriving.
	dispatchMaxBatch = 200 * time.Millisecond
)

// runDispatchWorker coalesces dispatch work over a short time window to avoid
// redundant Docker queries and broadcasts during bursts (e.g., compose up
// creating multiple containers). Events are collected, deduplicated by
// channel, and processed once per channel per batch.
func (app *App) runDispatchWorker(ctx context.Context) {
	for {
		// Block until the first event arrives.
		var first dispatchWork
		select {
		case <-ctx.Done():
			return
		case first = <-app.dispatchCh:
		}

		// Collect more events over a short window.
		fullSyncs := make(map[string]bool)
		var events []docker.DockerEvent

		if first.fullSync != "" {
			fullSyncs[first.fullSync] = true
		} else {
			events = append(events, first.evt)
		}

		quiet := time.NewTimer(dispatchQuietPeriod)
		deadline := time.NewTimer(dispatchMaxBatch)

	collect:
		for {
			select {
			case <-ctx.Done():
				quiet.Stop()
				deadline.Stop()
				return
			case work := <-app.dispatchCh:
				if work.fullSync != "" {
					fullSyncs[work.fullSync] = true
				} else {
					events = append(events, work.evt)
				}
				// Reset quiet timer on each new event
				if !quiet.Stop() {
					select {
					case <-quiet.C:
					default:
					}
				}
				quiet.Reset(dispatchQuietPeriod)
			case <-quiet.C:
				break collect
			case <-deadline.C:
				break collect
			}
		}

		quiet.Stop()
		deadline.Stop()

		// Process: track specific resource IDs to query (deduped via map).
		// Full syncs override filtered queries for that channel.
		affectedContainerIDs := make(map[string]bool)
		affectedNetworkIDs := make(map[string]bool)
		affectedImageIDs := make(map[string]bool)
		affectedVolumeNames := make(map[string]bool)

		var destroyedContainers []string
		var destroyedNetworks []string
		var destroyedImages []string
		var destroyedVolumes []string

		for _, evt := range events {
			switch evt.Type {
			case "container":
				if evt.ContainerID != "" {
					affectedContainerIDs[evt.ContainerID] = true
				}
				if evt.Action == "destroy" && evt.Name != "" {
					destroyedContainers = append(destroyedContainers, evt.Name)
				}
			case "network":
				if evt.Action == "connect" || evt.Action == "disconnect" {
					// Network connect/disconnect changes a container's network list
					if evt.ContainerID != "" {
						affectedContainerIDs[evt.ContainerID] = true
					} else {
						// Defensive: no container ID, fall back to full container sync
						fullSyncs[chanContainers] = true
					}
				} else {
					if evt.ActorID != "" {
						affectedNetworkIDs[evt.ActorID] = true
					}
					if evt.Action == "destroy" && evt.Name != "" {
						destroyedNetworks = append(destroyedNetworks, evt.Name)
					}
				}
			case "image":
				if evt.ActorID != "" {
					affectedImageIDs[evt.ActorID] = true
				}
				if evt.Action == "delete" && evt.ActorID != "" {
					destroyedImages = append(destroyedImages, evt.ActorID)
				}
			case "volume":
				if evt.Action == "mount" || evt.Action == "unmount" {
					// Volume mount/unmount changes a container's mount list
					if evt.ContainerID != "" {
						affectedContainerIDs[evt.ContainerID] = true
					} else {
						// Defensive: no container ID, fall back to full container sync
						fullSyncs[chanContainers] = true
					}
				} else {
					name := evt.Name
					if name == "" {
						name = evt.ActorID
					}
					if name != "" {
						affectedVolumeNames[name] = true
					}
					if evt.Action == "destroy" && name != "" {
						destroyedVolumes = append(destroyedVolumes, name)
					}
				}
			}
		}

		// Flush: filtered broadcasts for event-driven updates,
		// full-list broadcasts for explicit full-syncs.
		if len(affectedContainerIDs) > 0 || len(destroyedContainers) > 0 {
			app.BcastMetrics.recordTriggered(chanContainers)
			if fullSyncs[chanContainers] {
				app.broadcastContainersMap()
			} else {
				app.broadcastContainersByIDs(affectedContainerIDs, destroyedContainers)
			}
		} else if fullSyncs[chanContainers] {
			app.BcastMetrics.recordTriggered(chanContainers)
			app.broadcastContainersMap()
		}

		if len(affectedNetworkIDs) > 0 || len(destroyedNetworks) > 0 {
			app.BcastMetrics.recordTriggered(chanNetworks)
			if fullSyncs[chanNetworks] {
				app.broadcastNetworksMap()
			} else {
				app.broadcastNetworksByIDs(affectedNetworkIDs, destroyedNetworks)
			}
		} else if fullSyncs[chanNetworks] {
			app.BcastMetrics.recordTriggered(chanNetworks)
			app.broadcastNetworksMap()
		}

		if len(affectedImageIDs) > 0 || len(destroyedImages) > 0 {
			app.BcastMetrics.recordTriggered(chanImages)
			if fullSyncs[chanImages] {
				app.broadcastImagesMap()
			} else {
				app.broadcastImagesByIDs(affectedImageIDs, destroyedImages)
			}
		} else if fullSyncs[chanImages] {
			app.BcastMetrics.recordTriggered(chanImages)
			app.broadcastImagesMap()
		}

		if len(affectedVolumeNames) > 0 || len(destroyedVolumes) > 0 {
			app.BcastMetrics.recordTriggered(chanVolumes)
			if fullSyncs[chanVolumes] {
				app.broadcastVolumesMap()
			} else {
				app.broadcastVolumesByNames(affectedVolumeNames, destroyedVolumes)
			}
		} else if fullSyncs[chanVolumes] {
			app.BcastMetrics.recordTriggered(chanVolumes)
			app.broadcastVolumesMap()
		}

		// Handle remaining full-syncs (stacks, updates, etc.)
		for ch := range fullSyncs {
			if ch == chanContainers || ch == chanNetworks || ch == chanImages || ch == chanVolumes {
				continue // already handled above
			}
			app.BcastMetrics.recordTriggered(ch)
			app.dispatchFullSync(ctx, ch)
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

// runBroadcastWatcherLoop subscribes to Docker events and sends them to the
// dispatch channel. On error it retries with exponential backoff. The failure
// counter resets after a successful connection that lasts at least 30 seconds,
// so transient errors don't accumulate toward the limit across long uptimes.
func (app *App) runBroadcastWatcherLoop(ctx context.Context) {
	const maxConsecutiveFailures = 10
	failures := 0
	backoff := 1 * time.Second

	for {
		eventCh, errCh := app.Docker.Events(ctx)

		start := time.Now()
		err := app.consumeBroadcastEvents(ctx, eventCh, errCh)
		if ctx.Err() != nil {
			return // clean shutdown
		}

		// If the connection lasted a while, reset the failure counter —
		// this was a healthy connection that eventually broke, not a
		// connect-fail loop.
		if time.Since(start) > 30*time.Second {
			failures = 0
			backoff = 1 * time.Second
		}

		failures++
		if failures > maxConsecutiveFailures {
			slog.Error("docker events (broadcast): too many consecutive failures, backing off to max",
				"failures", failures, "lastErr", err)
			// Don't exit — keep retrying at max backoff. The daemon
			// may recover (e.g., Docker restart, socket reconnect).
			failures = maxConsecutiveFailures // cap to prevent overflow
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
// Each event is immediately broadcast on the "resourceEvent" channel for instant
// frontend notification; the dispatch worker handles authoritative list broadcasts.
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

			// Broadcast the event on the dedicated resourceEvent channel
			resEvt := toResourceEvent(evt)
			ws.BroadcastAuthenticated(app.WS, chanResourceEvent, resEvt)

			// Send to dispatch channel for authoritative list-based broadcast
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
