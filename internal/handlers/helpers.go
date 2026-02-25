package handlers

import (
    "encoding/json"
    "log/slog"
    "sync"
    "sync/atomic"

    "github.com/cfilipov/dockge/internal/compose"
    "github.com/cfilipov/dockge/internal/docker"
    "github.com/cfilipov/dockge/internal/models"
    "github.com/cfilipov/dockge/internal/terminal"
    "github.com/cfilipov/dockge/internal/ws"
)

// App holds shared dependencies for all handlers.
type App struct {
    Users        *models.UserStore
    Settings     *models.SettingStore
    Agents       *models.AgentStore
    ImageUpdates *models.ImageUpdateStore
    WS           *ws.Server
    Docker       docker.Client
    Compose      compose.Composer
    ComposeCache *compose.ComposeCache
    Terms        *terminal.Manager
    Mock         bool
    NoAuth       bool // Skip authentication checks (all endpoints open)

    JWTSecret        string
    NeedSetup        bool
    Version          string
    StacksDir        string
    MainTerminalName string // tracked for checkMainTerminal

    // recreateCache: stack name → true if any service needs recreation
    // (running image differs from compose.yaml image). Populated by serviceStatusList.
    // Uses copy-on-write via atomic.Pointer for zero-allocation reads.
    recreateMu       sync.Mutex              // protects writes to recreateInternal
    recreateInternal map[string]bool          // mutable, guarded by recreateMu
    recreateSnapshot atomic.Pointer[map[string]bool] // immutable snapshot for readers
}

// checkLogin verifies that the connection is authenticated.
// Returns the user ID or sends an error ack and returns 0.
// When --no-auth is enabled, connections are auto-authenticated at connect time,
// so this function returns 1 without any special handling.
func checkLogin(c *ws.Conn, msg *ws.ClientMessage) int {
    uid := c.UserID()
    if uid == 0 && msg.ID != nil {
        c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Not logged in"})
    }
    return uid
}

// parseArgs unmarshals the Args JSON array into a slice of json.RawMessage.
func parseArgs(msg *ws.ClientMessage) []json.RawMessage {
    if msg == nil || len(msg.Args) == 0 {
        return nil
    }
    var args []json.RawMessage
    if err := json.Unmarshal(msg.Args, &args); err != nil {
        slog.Warn("parse args", "err", err)
        return nil
    }
    return args
}

// argString extracts a string from args at the given index.
func argString(args []json.RawMessage, index int) string {
    if index >= len(args) {
        return ""
    }
    var s string
    if err := json.Unmarshal(args[index], &s); err != nil {
        return ""
    }
    return s
}

// argObject extracts a JSON object from args at the given index into dst.
func argObject(args []json.RawMessage, index int, dst interface{}) bool {
    if index >= len(args) {
        return false
    }
    return json.Unmarshal(args[index], dst) == nil
}

// argBool extracts a bool from args at the given index.
func argBool(args []json.RawMessage, index int) bool {
    if index >= len(args) {
        return false
    }
    var b bool
    if err := json.Unmarshal(args[index], &b); err != nil {
        return false
    }
    return b
}

// GetRecreateCache returns the immutable snapshot of the recreate cache.
// Zero allocations — just loads an atomic pointer.
// Callers must NOT mutate the returned map; it is shared.
func (app *App) GetRecreateCache() map[string]bool {
    if p := app.recreateSnapshot.Load(); p != nil {
        return *p
    }
    return map[string]bool{}
}

// SetRecreateNecessary updates the recreate flag for a stack and publishes
// a new immutable snapshot via atomic pointer (copy-on-write).
func (app *App) SetRecreateNecessary(stackName string, needed bool) {
    app.recreateMu.Lock()
    defer app.recreateMu.Unlock()
    if app.recreateInternal == nil {
        app.recreateInternal = make(map[string]bool)
    }
    app.recreateInternal[stackName] = needed

    // Build and publish immutable snapshot
    snap := make(map[string]bool, len(app.recreateInternal))
    for k, v := range app.recreateInternal {
        snap[k] = v
    }
    app.recreateSnapshot.Store(&snap)
}

// SetRecreateNecessaryBatch applies multiple recreate flag updates in a single
// lock acquisition and publishes one snapshot. Eliminates N intermediate snapshots
// when refreshing all stacks.
func (app *App) SetRecreateNecessaryBatch(updates map[string]bool) {
    app.recreateMu.Lock()
    defer app.recreateMu.Unlock()
    if app.recreateInternal == nil {
        app.recreateInternal = make(map[string]bool)
    }
    for k, v := range updates {
        app.recreateInternal[k] = v
    }

    // Build and publish immutable snapshot
    snap := make(map[string]bool, len(app.recreateInternal))
    for k, v := range app.recreateInternal {
        snap[k] = v
    }
    app.recreateSnapshot.Store(&snap)
}

// GetImageUpdateMap returns stack name → true for stacks with available updates.
func (app *App) GetImageUpdateMap() map[string]bool {
    m, err := app.ImageUpdates.StackHasUpdates()
    if err != nil {
        slog.Warn("image update map", "err", err)
        return map[string]bool{}
    }
    return m
}

// argInt extracts an integer from args at the given index.
func argInt(args []json.RawMessage, index int) int {
    if index >= len(args) {
        return 0
    }
    var n float64 // JSON numbers decode as float64
    if err := json.Unmarshal(args[index], &n); err != nil {
        return 0
    }
    return int(n)
}
