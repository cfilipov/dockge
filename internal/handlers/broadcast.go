package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"hash"
	"hash/fnv"
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

// channelDebouncer manages per-channel trailing-edge debounce timers.
// Each event type resets its own timer; the timer fires 200ms after the
// last event of that type.
type channelDebouncer struct {
	mu     sync.Mutex
	timers map[string]*time.Timer
}

func newChannelDebouncer() *channelDebouncer {
	return &channelDebouncer{
		timers: make(map[string]*time.Timer),
	}
}

// trigger resets the timer for the given channel. When the timer fires
// (200ms after the last trigger), it calls fn in a new goroutine.
func (d *channelDebouncer) trigger(channel string, fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if t, ok := d.timers[channel]; ok {
		t.Stop()
	}
	d.timers[channel] = time.AfterFunc(200*time.Millisecond, fn)
}

// broadcastState holds per-channel FNV hashes for deduplication.
type broadcastState struct {
	mu       sync.Mutex
	lastHash map[string]uint64
	hasher   hash.Hash64
}

func newBroadcastState() *broadcastState {
	return &broadcastState{
		lastHash: make(map[string]uint64),
		hasher:   fnv.New64a(),
	}
}

// BroadcastMetrics tracks per-channel broadcast statistics.
type BroadcastMetrics struct {
	mu       sync.Mutex
	counters map[string]*ChannelMetrics
}

// ChannelMetrics holds counters for a single broadcast channel.
type ChannelMetrics struct {
	Triggered int64 `json:"triggered"` // debouncer triggered
	Sent      int64 `json:"sent"`      // actually broadcast (hash changed)
	Skipped   int64 `json:"skipped"`   // suppressed by hash dedup
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

func (bm *BroadcastMetrics) recordSkipped(channel string) {
	bm.mu.Lock()
	if cm, ok := bm.counters[channel]; ok {
		cm.Skipped++
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
			Skipped:   v.Skipped,
		}
	}
	return result
}

// broadcastIfChanged marshals data, computes FNV-1a hash, and broadcasts
// to all authenticated connections only if the hash differs from the last
// broadcast on this channel. Returns true if a broadcast was sent.
func (bs *broadcastState) broadcastIfChanged(wss *ws.Server, channel string, data any, metrics *BroadcastMetrics) bool {
	// Marshal the full envelope once — used for both hashing and sending.
	msg, err := json.Marshal(ws.ServerMessage[any]{
		Event: channel,
		Data:  data,
	})
	if err != nil {
		slog.Error("broadcast marshal", "channel", channel, "err", err)
		return false
	}

	bs.hasher.Reset()
	bs.hasher.Write(msg)
	hash := bs.hasher.Sum64()

	bs.mu.Lock()
	old := bs.lastHash[channel]
	changed := hash != old
	if changed {
		bs.lastHash[channel] = hash
	}
	bs.mu.Unlock()

	if !changed {
		slog.Debug("broadcast skipped (unchanged)", "channel", channel)
		if metrics != nil {
			metrics.recordSkipped(channel)
		}
		return false
	}

	wss.BroadcastAuthenticatedBytes(msg)
	slog.Info("broadcast sent", "channel", channel, "bytes", len(msg))
	if metrics != nil {
		metrics.recordSent(channel)
	}
	return true
}

// sendToConn sends channel data to a single connection (used for initial connect).
func sendToConn(c *ws.Conn, channel string, data any) {
	ws.SendEvent(c, channel, data)
}

// broadcastStacks scans the stacks directory and broadcasts compose file metadata.
func (app *App) broadcastStacks() {
	if !app.WS.HasAuthenticatedConns() {
		return
	}
	entries := buildStackBroadcast(app.StacksDir)
	app.bcastState.broadcastIfChanged(app.WS, chanStacks, entries, app.BcastMetrics)
}

// broadcastContainers queries Docker for all containers and broadcasts enriched data.
func (app *App) broadcastContainers() {
	if !app.WS.HasAuthenticatedConns() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	containers, err := app.Docker.ContainerListDetailed(ctx)
	if err != nil {
		slog.Warn("broadcastContainers", "err", err)
		containers = []docker.ContainerBroadcast{}
	}
	app.bcastState.broadcastIfChanged(app.WS, chanContainers, containers, app.BcastMetrics)
}

// broadcastNetworks queries Docker for all networks and broadcasts.
func (app *App) broadcastNetworks() {
	if !app.WS.HasAuthenticatedConns() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	networks, err := app.Docker.NetworkList(ctx)
	if err != nil {
		slog.Warn("broadcastNetworks", "err", err)
		networks = []docker.NetworkSummary{}
	}
	app.bcastState.broadcastIfChanged(app.WS, chanNetworks, networks, app.BcastMetrics)
}

// broadcastImages queries Docker for all images and broadcasts.
func (app *App) broadcastImages() {
	if !app.WS.HasAuthenticatedConns() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	images, err := app.Docker.ImageList(ctx)
	if err != nil {
		slog.Warn("broadcastImages", "err", err)
		images = []docker.ImageSummary{}
	}
	app.bcastState.broadcastIfChanged(app.WS, chanImages, images, app.BcastMetrics)
}

// broadcastVolumes queries Docker for all volumes and broadcasts.
func (app *App) broadcastVolumes() {
	if !app.WS.HasAuthenticatedConns() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	volumes, err := app.Docker.VolumeList(ctx)
	if err != nil {
		slog.Warn("broadcastVolumes", "err", err)
		volumes = []docker.VolumeSummary{}
	}
	app.bcastState.broadcastIfChanged(app.WS, chanVolumes, volumes, app.BcastMetrics)
}

// broadcastUpdates reads BoltDB image update cache and broadcasts container names with updates.
func (app *App) broadcastUpdates() {
	svcUpdates, err := app.ImageUpdates.AllServiceUpdates()
	if err != nil {
		slog.Warn("broadcastUpdates", "err", err)
		svcUpdates = map[string]bool{}
	}

	// Collect service keys that have updates
	updated := make([]string, 0, len(svcUpdates))
	for key, hasUpdate := range svcUpdates {
		if hasUpdate {
			updated = append(updated, key)
		}
	}
	sort.Strings(updated)

	app.bcastState.broadcastIfChanged(app.WS, chanUpdates, updated, app.BcastMetrics)
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

	// Sort by name for deterministic serialization
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// InitBroadcast initializes the broadcast state and debouncer. Must be called
// before StartBroadcastWatcher or any broadcast trigger methods.
func (app *App) InitBroadcast() {
	app.bcastState = newBroadcastState()
	app.debouncer = newChannelDebouncer()
	app.BcastMetrics = newBroadcastMetrics()
	app.EventBus = NewEventBus()
}

// StartBroadcastWatcher starts the event-driven broadcast system.
// It subscribes to Docker events and dispatches to per-channel broadcast
// functions via the debouncer. On error it retries with exponential backoff;
// after repeated failures it exits the process.
//
// The watcher starts once at boot and never stops — client guards in
// individual broadcast functions skip Docker API calls when no clients
// are connected.
func (app *App) StartBroadcastWatcher(ctx context.Context) {
	slog.Info("broadcast watcher started")
	go app.runBroadcastWatcherLoop(ctx)
}

// runBroadcastWatcherLoop subscribes to Docker events and dispatches to
// per-channel broadcasters. On error or channel close, it retries with
// exponential backoff up to maxRetries times, then exits the process.
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

// consumeBroadcastEvents reads Docker events and dispatches to per-channel
// broadcasters until the channel closes or errors.
func (app *App) consumeBroadcastEvents(ctx context.Context, eventCh <-chan docker.DockerEvent, errCh <-chan error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case evt, ok := <-eventCh:
			if !ok {
				return fmt.Errorf("docker events channel closed")
			}
			slog.Debug("docker event", "type", evt.Type, "action", evt.Action)

			// Fan out to per-terminal subscribers (always, even without
			// authenticated WS conns — terminals have their own lifecycle).
			app.EventBus.Publish(evt)

			if !app.WS.HasAuthenticatedConns() {
				continue
			}

			// Broadcast raw Docker event for dev inspection (visible in browser console).
			if evt.Raw != nil {
				ws.BroadcastAuthenticated(app.WS, "dockerEvent", evt.Raw)
			} else {
				ws.BroadcastAuthenticated(app.WS, "dockerEvent", evt)
			}

			switch evt.Type {
			case "container":
				app.BcastMetrics.recordTriggered(chanContainers)
				app.debouncer.trigger(chanContainers, app.broadcastContainers)
			case "network":
				app.BcastMetrics.recordTriggered(chanNetworks)
				app.debouncer.trigger(chanNetworks, app.broadcastNetworks)
			case "image":
				app.BcastMetrics.recordTriggered(chanImages)
				app.debouncer.trigger(chanImages, app.broadcastImages)
			case "volume":
				app.BcastMetrics.recordTriggered(chanVolumes)
				app.debouncer.trigger(chanVolumes, app.broadcastVolumes)
			}

		case err, ok := <-errCh:
			if !ok {
				continue
			}
			return fmt.Errorf("docker events error: %w", err)
		}
	}
}

// TriggerStacksBroadcast triggers a debounced stacks broadcast (used by fsnotify watcher).
func (app *App) TriggerStacksBroadcast() {
	app.debouncer.trigger(chanStacks, app.broadcastStacks)
}

// TriggerContainersBroadcast triggers a debounced containers broadcast.
func (app *App) TriggerContainersBroadcast() {
	app.debouncer.trigger(chanContainers, app.broadcastContainers)
}

// TriggerNetworksBroadcast triggers a debounced networks broadcast.
func (app *App) TriggerNetworksBroadcast() {
	app.debouncer.trigger(chanNetworks, app.broadcastNetworks)
}

// TriggerImagesBroadcast triggers a debounced images broadcast.
func (app *App) TriggerImagesBroadcast() {
	app.debouncer.trigger(chanImages, app.broadcastImages)
}

// TriggerVolumesBroadcast triggers a debounced volumes broadcast.
func (app *App) TriggerVolumesBroadcast() {
	app.debouncer.trigger(chanVolumes, app.broadcastVolumes)
}

// TriggerUpdatesBroadcast triggers a debounced updates broadcast.
func (app *App) TriggerUpdatesBroadcast() {
	app.debouncer.trigger(chanUpdates, app.broadcastUpdates)
}
