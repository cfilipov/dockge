package handlers

import (
	"context"
	"encoding/json"
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

// stop cancels all pending timers.
func (d *channelDebouncer) stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, t := range d.timers {
		t.Stop()
	}
}

// broadcastState holds per-channel FNV hashes for deduplication.
type broadcastState struct {
	mu       sync.Mutex
	lastHash map[string]uint64
}

func newBroadcastState() *broadcastState {
	return &broadcastState{
		lastHash: make(map[string]uint64),
	}
}

// broadcastIfChanged marshals data, computes FNV-1a hash, and broadcasts
// to all authenticated connections only if the hash differs from the last
// broadcast on this channel. Returns true if a broadcast was sent.
func (bs *broadcastState) broadcastIfChanged(wss *ws.Server, channel string, data interface{}) bool {
	payload, err := json.Marshal(data)
	if err != nil {
		slog.Error("broadcast marshal", "channel", channel, "err", err)
		return false
	}

	h := fnv.New64a()
	h.Write(payload)
	hash := h.Sum64()

	bs.mu.Lock()
	old := bs.lastHash[channel]
	changed := hash != old
	if changed {
		bs.lastHash[channel] = hash
	}
	bs.mu.Unlock()

	if !changed {
		slog.Debug("broadcast skipped (unchanged)", "channel", channel)
		return false
	}

	// Wrap in the standard server message format
	msg, err := json.Marshal(ws.ServerMessage{
		Event: channel,
		Data:  data,
	})
	if err != nil {
		slog.Error("broadcast wrap", "channel", channel, "err", err)
		return false
	}

	wss.BroadcastAuthenticatedBytes(msg)
	slog.Debug("broadcast sent", "channel", channel, "bytes", len(msg))
	return true
}

// sendToConn sends channel data to a single connection (used for initial connect).
func sendToConn(c *ws.Conn, channel string, data interface{}) {
	c.SendEvent(channel, data)
}

// broadcastStacks scans the stacks directory and broadcasts compose file metadata.
func (app *App) broadcastStacks() {
	entries := buildStackBroadcast(app.StacksDir)
	app.bcastState.broadcastIfChanged(app.WS, chanStacks, entries)
}

// broadcastContainers queries Docker for all containers and broadcasts enriched data.
func (app *App) broadcastContainers() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	containers, err := app.Docker.ContainerBroadcastList(ctx)
	if err != nil {
		slog.Warn("broadcastContainers", "err", err)
		containers = []docker.ContainerBroadcast{}
	}
	app.bcastState.broadcastIfChanged(app.WS, chanContainers, containers)
}

// broadcastNetworks queries Docker for all networks and broadcasts.
func (app *App) broadcastNetworks() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	networks, err := app.Docker.NetworkList(ctx)
	if err != nil {
		slog.Warn("broadcastNetworks", "err", err)
		networks = []docker.NetworkSummary{}
	}
	app.bcastState.broadcastIfChanged(app.WS, chanNetworks, networks)
}

// broadcastImages queries Docker for all images and broadcasts.
func (app *App) broadcastImages() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	images, err := app.Docker.ImageList(ctx)
	if err != nil {
		slog.Warn("broadcastImages", "err", err)
		images = []docker.ImageSummary{}
	}
	app.bcastState.broadcastIfChanged(app.WS, chanImages, images)
}

// broadcastVolumes queries Docker for all volumes and broadcasts.
func (app *App) broadcastVolumes() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	volumes, err := app.Docker.VolumeList(ctx)
	if err != nil {
		slog.Warn("broadcastVolumes", "err", err)
		volumes = []docker.VolumeSummary{}
	}
	app.bcastState.broadcastIfChanged(app.WS, chanVolumes, volumes)
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

	app.bcastState.broadcastIfChanged(app.WS, chanUpdates, updated)
}

// sendAllBroadcastsTo sends current state of all 6 channels to a single connection.
// Used on initial authenticated connect.
func (app *App) sendAllBroadcastsTo(c *ws.Conn) {
	// 1. Stacks (fastest — dir scan + compose parse)
	entries := buildStackBroadcast(app.StacksDir)
	sendToConn(c, chanStacks, entries)

	// 2. Containers
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	containers, err := app.Docker.ContainerBroadcastList(ctx)
	if err != nil {
		slog.Warn("sendAllBroadcastsTo: containers", "err", err)
		containers = []docker.ContainerBroadcast{}
	}
	sendToConn(c, chanContainers, containers)

	// 3-5. Networks, Images, Volumes (parallel)
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		networks, err := app.Docker.NetworkList(ctx)
		if err != nil {
			slog.Warn("sendAllBroadcastsTo: networks", "err", err)
			networks = []docker.NetworkSummary{}
		}
		sendToConn(c, chanNetworks, networks)
	}()
	go func() {
		defer wg.Done()
		images, err := app.Docker.ImageList(ctx)
		if err != nil {
			slog.Warn("sendAllBroadcastsTo: images", "err", err)
			images = []docker.ImageSummary{}
		}
		sendToConn(c, chanImages, images)
	}()
	go func() {
		defer wg.Done()
		volumes, err := app.Docker.VolumeList(ctx)
		if err != nil {
			slog.Warn("sendAllBroadcastsTo: volumes", "err", err)
			volumes = []docker.VolumeSummary{}
		}
		sendToConn(c, chanVolumes, volumes)
	}()
	wg.Wait()

	// 6. Updates (BoltDB read — instant)
	svcUpdates, _ := app.ImageUpdates.AllServiceUpdates()
	updated := make([]string, 0, len(svcUpdates))
	for key, hasUpdate := range svcUpdates {
		if hasUpdate {
			updated = append(updated, key)
		}
	}
	sort.Strings(updated)
	sendToConn(c, chanUpdates, updated)
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

// InitBroadcast initializes the broadcast state. Must be called before
// StartBroadcastWatcher or any broadcast trigger methods.
func (app *App) InitBroadcast() {
	app.bcastState = newBroadcastState()
}

// StartBroadcastWatcher starts the event-driven broadcast system.
// It subscribes to Docker events and dispatches to per-channel broadcast
// functions via a debouncer. Also starts the fsnotify watcher for stacks.
func (app *App) StartBroadcastWatcher(ctx context.Context) {
	debouncer := newChannelDebouncer()
	app.debouncer = debouncer

	// Initial broadcast of all channels
	if app.WS.HasAuthenticatedConns() {
		app.broadcastStacks()
		app.broadcastContainers()
		app.broadcastNetworks()
		app.broadcastImages()
		app.broadcastVolumes()
		app.broadcastUpdates()
	}

	// Subscribe to Docker events
	eventCh, errCh := app.Docker.Events(ctx)

	go func() {
		defer debouncer.stop()

		for {
			select {
			case <-ctx.Done():
				return

			case evt, ok := <-eventCh:
				if !ok {
					slog.Warn("docker events channel closed, falling back to polling")
					app.runBroadcastPollingFallback(ctx, debouncer)
					return
				}
				slog.Debug("docker event", "type", evt.Type, "action", evt.Action)

				if !app.WS.HasAuthenticatedConns() {
					continue
				}

				// Dispatch to the appropriate channel's debouncer
				switch evt.Type {
				case "container":
					debouncer.trigger(chanContainers, app.broadcastContainers)
				case "network":
					debouncer.trigger(chanNetworks, app.broadcastNetworks)
				case "image":
					debouncer.trigger(chanImages, app.broadcastImages)
				case "volume":
					debouncer.trigger(chanVolumes, app.broadcastVolumes)
				}

			case err, ok := <-errCh:
				if !ok {
					continue
				}
				slog.Warn("docker events error", "err", err)
				app.runBroadcastPollingFallback(ctx, debouncer)
				return
			}
		}
	}()
}

// runBroadcastPollingFallback polls all channels every 60s when events are unavailable.
func (app *App) runBroadcastPollingFallback(ctx context.Context, debouncer *channelDebouncer) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !app.WS.HasAuthenticatedConns() {
				continue
			}
			app.broadcastContainers()
			app.broadcastNetworks()
			app.broadcastImages()
			app.broadcastVolumes()
		}
	}
}

// TriggerStacksBroadcast triggers a debounced stacks broadcast (used by fsnotify watcher).
func (app *App) TriggerStacksBroadcast() {
	if app.debouncer != nil {
		app.debouncer.trigger(chanStacks, app.broadcastStacks)
	}
}

// TriggerContainersBroadcast triggers a debounced containers broadcast.
func (app *App) TriggerContainersBroadcast() {
	if app.debouncer != nil {
		app.debouncer.trigger(chanContainers, app.broadcastContainers)
	}
}

// TriggerAllBroadcasts triggers debounced broadcasts on all Docker channels.
// Used after compose operations (up, down, etc.) that may change multiple resource types.
func (app *App) TriggerAllBroadcasts() {
	if app.debouncer == nil {
		return
	}
	app.debouncer.trigger(chanContainers, app.broadcastContainers)
	app.debouncer.trigger(chanNetworks, app.broadcastNetworks)
	app.debouncer.trigger(chanImages, app.broadcastImages)
	app.debouncer.trigger(chanVolumes, app.broadcastVolumes)
}

// TriggerUpdatesBroadcast triggers a debounced updates broadcast.
func (app *App) TriggerUpdatesBroadcast() {
	if app.debouncer != nil {
		app.debouncer.trigger(chanUpdates, app.broadcastUpdates)
	}
}
