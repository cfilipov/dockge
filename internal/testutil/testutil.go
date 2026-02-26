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

    "github.com/cfilipov/dockge/internal/db"
    "github.com/cfilipov/dockge/internal/docker"
    "github.com/cfilipov/dockge/internal/handlers"
    "github.com/cfilipov/dockge/internal/models"
    "github.com/cfilipov/dockge/internal/terminal"
    "github.com/cfilipov/dockge/internal/ws"

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
    State     *docker.MockState
    cancel    context.CancelFunc
}

// Setup creates a test environment with a real HTTP server, BoltDB, and mock Docker.
// Only copies "test-stack" from test-data — fast for unit tests.
func Setup(t testing.TB) *TestEnv {
    return setupWithStacks(t, "test-stack")
}

// SetupFull creates a test environment with all 200+ stacks from test-data.
// Use only for stress tests and benchmarks.
func SetupFull(t testing.TB) *TestEnv {
    return setupWithStacks(t)
}

// setupWithStacks creates a test env. If stackNames is empty, copies all stacks;
// otherwise only the named ones.
func setupWithStacks(t testing.TB, stackNames ...string) *TestEnv {
    t.Helper()

    // Create temp directories
    tmpDir := t.TempDir()
    stacksDir := filepath.Join(tmpDir, "stacks")
    dataDir := filepath.Join(tmpDir, "data")

    if err := os.MkdirAll(stacksDir, 0755); err != nil {
        t.Fatal(err)
    }

    // Copy test stacks from test-data
    testdataDir := findTestdata(t)
    srcStacks := filepath.Join(testdataDir, "stacks")
    if len(stackNames) == 0 {
        // Copy everything
        copyDir(t, srcStacks, stacksDir)
    } else {
        // Copy only named stacks
        for _, name := range stackNames {
            src := filepath.Join(srcStacks, name)
            dst := filepath.Join(stacksDir, name)
            if err := os.MkdirAll(dst, 0755); err != nil {
                t.Fatal(err)
            }
            copyDir(t, src, dst)
        }
    }

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

    // Start fake Docker daemon with shared in-memory state
    state := docker.NewMockState()
    data := docker.BuildMockData(stacksDir)
    sockPath, daemonCleanup, err := docker.StartFakeDaemon(state, data, stacksDir)
    if err != nil {
        t.Fatal("start fake daemon:", err)
    }

    // Set DOCKER_HOST so SDKClient connects to fake daemon
    os.Setenv("DOCKER_HOST", "unix://"+sockPath)

    dockerClient, err := docker.NewSDKClient()
    if err != nil {
        daemonCleanup()
        t.Fatal("new sdk client:", err)
    }

    // Terminal manager
    terms := terminal.NewManager()

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
        Terms:        terms,
        JWTSecret:    jwtSecret,
        NeedSetup:    userCount == 0,
        Version:      "test",
        StacksDir:    stacksDir,
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
        dockerClient.Close()
        daemonCleanup()
        database.Close()
        os.Unsetenv("DOCKER_HOST")
    })

    return &TestEnv{
        App:       app,
        Server:    server,
        WSServer:  wss,
        StacksDir: stacksDir,
        DataDir:   dataDir,
        State:     state,
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

// SetStackRunning marks a stack as running in the mock state.
func (e *TestEnv) SetStackRunning(t testing.TB, stackName string) {
    t.Helper()
    e.State.Set(stackName, "running")
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

// findTestdata locates the test-data directory relative to the project root.
func findTestdata(t testing.TB) string {
    t.Helper()

    // Walk up from cwd to find test-data/
    dir, err := os.Getwd()
    if err != nil {
        t.Fatal(err)
    }

    for {
        candidate := filepath.Join(dir, "test-data")
        if info, err := os.Stat(candidate); err == nil && info.IsDir() {
            return candidate
        }
        parent := filepath.Dir(dir)
        if parent == dir {
            t.Fatal("test-data directory not found")
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
