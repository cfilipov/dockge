package handlers

import (
    "encoding/json"
    "log/slog"
    "sync"

    "github.com/cfilipov/dockge/backend-go/internal/compose"
    "github.com/cfilipov/dockge/backend-go/internal/models"
    "github.com/cfilipov/dockge/backend-go/internal/terminal"
    "github.com/cfilipov/dockge/backend-go/internal/ws"
)

// App holds shared dependencies for all handlers.
type App struct {
    Users        *models.UserStore
    Settings     *models.SettingStore
    Agents       *models.AgentStore
    ImageUpdates *models.ImageUpdateStore
    WS           *ws.Server
    Compose      *compose.Exec
    Terms        *terminal.Manager

    JWTSecret        string
    NeedSetup        bool
    Version          string
    StacksDir        string
    MainTerminalName string // tracked for checkMainTerminal

    // recreateCache: stack name → true if any service needs recreation
    // (running image differs from compose.yaml image). Populated by serviceStatusList.
    recreateMu    sync.RWMutex
    RecreateCache map[string]bool
}

// checkLogin verifies that the connection is authenticated.
// Returns the user ID or sends an error ack and returns 0.
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

// GetRecreateCache returns a snapshot of the recreate necessary cache.
func (app *App) GetRecreateCache() map[string]bool {
    app.recreateMu.RLock()
    defer app.recreateMu.RUnlock()
    if app.RecreateCache == nil {
        return map[string]bool{}
    }
    cp := make(map[string]bool, len(app.RecreateCache))
    for k, v := range app.RecreateCache {
        cp[k] = v
    }
    return cp
}

// SetRecreateNecessary updates the recreate flag for a stack.
func (app *App) SetRecreateNecessary(stackName string, needed bool) {
    app.recreateMu.Lock()
    defer app.recreateMu.Unlock()
    if app.RecreateCache == nil {
        app.RecreateCache = make(map[string]bool)
    }
    app.RecreateCache[stackName] = needed
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
