package testutil

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "os"
    "path/filepath"
    "sync/atomic"
    "testing"
    "time"

    "github.com/cfilipov/dockge/backend-go/internal/compose"
    "github.com/cfilipov/dockge/backend-go/internal/db"
    "github.com/cfilipov/dockge/backend-go/internal/docker"
    "github.com/cfilipov/dockge/backend-go/internal/handlers"
    "github.com/cfilipov/dockge/backend-go/internal/models"
    "github.com/cfilipov/dockge/backend-go/internal/terminal"
    "github.com/cfilipov/dockge/backend-go/internal/ws"

    "github.com/coder/websocket"
)

var msgIDCounter int64

// TestEnv holds a fully wired test application with temp DB and mock Docker.
type TestEnv struct {
    App       *handlers.App
    Server    *httptest.Server
    WSServer  *ws.Server
    StacksDir string
    DataDir   string
    StateDir  string
    cancel    context.CancelFunc
}

// Setup creates a test environment with a real HTTP server, BoltDB, and mock Docker.
// The test stacks from testdata/stacks/ are copied into a temp directory.
func Setup(t testing.TB) *TestEnv {
    t.Helper()

    // Create temp directories
    tmpDir := t.TempDir()
    stacksDir := filepath.Join(tmpDir, "stacks")
    dataDir := filepath.Join(tmpDir, "data")
    stateDir := filepath.Join(tmpDir, "mock-state")

    if err := os.MkdirAll(stacksDir, 0755); err != nil {
        t.Fatal(err)
    }
    if err := os.MkdirAll(stateDir, 0755); err != nil {
        t.Fatal(err)
    }

    // Copy test stacks from testdata
    testdataDir := findTestdata(t)
    copyDir(t, filepath.Join(testdataDir, "stacks"), stacksDir)

    // Open BoltDB in temp dir
    database, err := db.Open(dataDir)
    if err != nil {
        t.Fatal(err)
    }

    // Create stores
    users := models.NewUserStore(database)
    settings := models.NewSettingStore(database)
    agents := models.NewAgentStore(database)
    imageUpdates := models.NewImageUpdateStore(database)

    // Ensure JWT secret
    jwtSecret, err := settings.EnsureJWTSecret()
    if err != nil {
        t.Fatal(err)
    }

    // Check if setup needed
    userCount, err := users.Count()
    if err != nil {
        t.Fatal(err)
    }

    // Mock Docker + Compose with isolated state dir
    dockerClient := docker.NewMockClientWithStateDir(stacksDir, stateDir)
    composeExec := compose.NewMockComposeWithStateDir(stacksDir, stateDir)

    // Terminal manager
    terms := terminal.NewManager()

    // Compose cache
    composeCache := compose.NewComposeCache()
    composeCache.PopulateFromDisk(stacksDir)

    // WebSocket server
    wss := ws.NewServer()

    // Assemble App
    app := &handlers.App{
        Users:        users,
        Settings:     settings,
        Agents:       agents,
        ImageUpdates: imageUpdates,
        WS:           wss,
        Docker:       dockerClient,
        Compose:      composeExec,
        ComposeCache: composeCache,
        Terms:        terms,
        JWTSecret:    jwtSecret,
        NeedSetup:    userCount == 0,
        Version:      "test",
        StacksDir:    stacksDir,
        Mock:         true,
    }

    // Register all handlers
    handlers.RegisterAuthHandlers(app)
    handlers.RegisterSettingsHandlers(app)
    handlers.RegisterAgentHandlers(app)
    handlers.RegisterStackHandlers(app)
    handlers.RegisterTerminalHandlers(app)
    handlers.RegisterDockerHandlers(app)
    handlers.RegisterServiceHandlers(app)

    // Wire disconnect cleanup
    wss.OnDisconnect(func(c *ws.Conn) {
        terms.RemoveWriterFromAll(c.ID())
    })

    // HTTP mux with WS and health
    mux := http.NewServeMux()
    mux.Handle("/ws", wss.UpgradeHandler())
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("ok"))
    })

    // Start background tasks
    ctx, cancel := context.WithCancel(context.Background())
    app.StartStackWatcher(ctx)

    // Start test server
    server := httptest.NewServer(mux)

    t.Cleanup(func() {
        cancel()
        server.Close()
        database.Close()
    })

    return &TestEnv{
        App:       app,
        Server:    server,
        WSServer:  wss,
        StacksDir: stacksDir,
        DataDir:   dataDir,
        StateDir:  stateDir,
        cancel:    cancel,
    }
}

// SeedAdmin creates the admin user for tests that need authentication.
func (e *TestEnv) SeedAdmin(t testing.TB) {
    t.Helper()
    _, err := e.App.Users.Create("admin", "testpass123")
    if err != nil {
        t.Fatal("seed admin:", err)
    }
    e.App.NeedSetup = false
}

// SetStackRunning marks a stack as running in the mock state directory.
func (e *TestEnv) SetStackRunning(t testing.TB, stackName string) {
    t.Helper()
    stateFile := filepath.Join(e.StateDir, stackName, "status")
    if err := os.MkdirAll(filepath.Dir(stateFile), 0755); err != nil {
        t.Fatal(err)
    }
    if err := os.WriteFile(stateFile, []byte("running"), 0644); err != nil {
        t.Fatal(err)
    }
}

// DialWS opens a WebSocket connection to the test server.
// Push messages sent on connect (info, setup) are not drained here —
// SendAndReceive skips non-ack messages automatically.
func (e *TestEnv) DialWS(t testing.TB) *websocket.Conn {
    t.Helper()
    wsURL := "ws" + e.Server.URL[4:] + "/ws" // http -> ws
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    conn, _, err := websocket.Dial(ctx, wsURL, nil)
    if err != nil {
        t.Fatal("dial ws:", err)
    }
    conn.SetReadLimit(1 << 20)

    t.Cleanup(func() {
        conn.Close(websocket.StatusNormalClosure, "")
    })

    return conn
}

// Login sends a login event and waits for the ack with a JWT token.
// Returns the token string.
func (e *TestEnv) Login(t testing.TB, conn *websocket.Conn) string {
    t.Helper()
    resp := e.SendAndReceive(t, conn, "login", "admin", "testpass123", "", "")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("login failed: %v", resp)
    }
    token, _ := resp["token"].(string)
    return token
}

// SendAndReceive sends a WS event with an ack ID and returns the parsed ack response.
func (e *TestEnv) SendAndReceive(t testing.TB, conn *websocket.Conn, event string, args ...interface{}) map[string]interface{} {
    t.Helper()

    id := atomic.AddInt64(&msgIDCounter, 1)

    argsJSON, err := json.Marshal(args)
    if err != nil {
        t.Fatal("marshal args:", err)
    }

    msg := map[string]interface{}{
        "id":    id,
        "event": event,
        "args":  json.RawMessage(argsJSON),
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    data, err := json.Marshal(msg)
    if err != nil {
        t.Fatal("marshal msg:", err)
    }

    if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
        t.Fatal("write:", err)
    }

    // Read messages until we find our ack
    for {
        _, respData, err := conn.Read(ctx)
        if err != nil {
            t.Fatal("read:", err)
        }

        var raw map[string]json.RawMessage
        if err := json.Unmarshal(respData, &raw); err != nil {
            t.Fatal("unmarshal response:", err)
        }

        // Check if this is an ack (has "id" field)
        if idRaw, ok := raw["id"]; ok {
            var ackID int64
            if err := json.Unmarshal(idRaw, &ackID); err == nil && ackID == id {
                // This is our ack — parse the data field
                var ack struct {
                    Data map[string]interface{} `json:"data"`
                }
                if err := json.Unmarshal(respData, &ack); err != nil {
                    t.Fatal("unmarshal ack:", err)
                }
                return ack.Data
            }
        }
        // Not our ack — it's a push message, skip it
    }
}

// SendEvent sends a WS event without waiting for an ack.
func (e *TestEnv) SendEvent(t testing.TB, conn *websocket.Conn, event string, args ...interface{}) {
    t.Helper()

    argsJSON, err := json.Marshal(args)
    if err != nil {
        t.Fatal("marshal args:", err)
    }

    msg := map[string]interface{}{
        "event": event,
        "args":  json.RawMessage(argsJSON),
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    data, err := json.Marshal(msg)
    if err != nil {
        t.Fatal("marshal msg:", err)
    }

    if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
        t.Fatal("write:", err)
    }
}

// findTestdata locates the testdata directory relative to the backend-go root.
func findTestdata(t testing.TB) string {
    t.Helper()

    // Walk up from cwd to find testdata/
    dir, err := os.Getwd()
    if err != nil {
        t.Fatal(err)
    }

    for {
        candidate := filepath.Join(dir, "testdata")
        if info, err := os.Stat(candidate); err == nil && info.IsDir() {
            return candidate
        }
        parent := filepath.Dir(dir)
        if parent == dir {
            t.Fatal("testdata directory not found")
        }
        dir = parent
    }
}

// copyDir recursively copies src to dst.
func copyDir(t testing.TB, src, dst string) {
    t.Helper()
    entries, err := os.ReadDir(src)
    if err != nil {
        t.Fatal("read dir:", err)
    }

    for _, entry := range entries {
        srcPath := filepath.Join(src, entry.Name())
        dstPath := filepath.Join(dst, entry.Name())

        if entry.IsDir() {
            if err := os.MkdirAll(dstPath, 0755); err != nil {
                t.Fatal(err)
            }
            copyDir(t, srcPath, dstPath)
        } else {
            data, err := os.ReadFile(srcPath)
            if err != nil {
                t.Fatal("read file:", err)
            }
            if err := os.WriteFile(dstPath, data, 0644); err != nil {
                t.Fatal("write file:", err)
            }
        }
    }
}
