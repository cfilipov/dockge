package testutil

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "net/http/httptest"
    "os"
    "os/exec"
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

// daemon holds the singleton mock daemon process state.
var daemon struct {
    proc      *os.Process
    stacksDir string // the --stacks-dir the daemon is serving
    tmpDir    string // temp dir containing socket + stacks
}

// StartDaemon launches the mock daemon binary on a temp Unix socket,
// sets DOCKER_HOST to point to it, and prepends dockerCLI's directory
// to PATH. Call this once from TestMain before m.Run().
//
// daemonBin: path to the compiled mock-daemon binary
// dockerCLI: path to the compiled mock docker CLI binary
func StartDaemon(daemonBin, dockerCLI string) {
    tmpDir, err := os.MkdirTemp("", "dockge-test-*")
    if err != nil {
        log.Fatalf("testutil: create temp dir: %v", err)
    }
    daemon.tmpDir = tmpDir

    sockPath := filepath.Join(tmpDir, "docker.sock")
    stacksDir := filepath.Join(tmpDir, "stacks")
    daemon.stacksDir = stacksDir

    // Locate test-data/stacks and images.json relative to project root
    stacksSource := findStacksDirFatal()
    imagesJSON := filepath.Join(filepath.Dir(stacksSource), "..", "mock-docker", "scripts", "images.json")

    cmd := exec.Command(daemonBin,
        "--socket", sockPath,
        "--stacks-source", stacksSource,
        "--stacks-dir", stacksDir,
        "--images-json", imagesJSON,
    )
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    if err := cmd.Start(); err != nil {
        os.RemoveAll(tmpDir)
        log.Fatalf("testutil: start mock daemon: %v", err)
    }
    daemon.proc = cmd.Process

    // Wait for socket to appear (up to 5s)
    for i := 0; i < 50; i++ {
        if _, err := os.Stat(sockPath); err == nil {
            break
        }
        time.Sleep(100 * time.Millisecond)
    }
    if _, err := os.Stat(sockPath); err != nil {
        daemon.proc.Kill()
        os.RemoveAll(tmpDir)
        log.Fatalf("testutil: mock daemon socket not ready after 5s")
    }

    os.Setenv("DOCKER_HOST", "unix://"+sockPath)

    // Prepend mock docker CLI directory to PATH
    cliDir := filepath.Dir(dockerCLI)
    if abs, err := filepath.Abs(cliDir); err == nil {
        cliDir = abs
    }
    os.Setenv("PATH", cliDir+":"+os.Getenv("PATH"))
}

// StopDaemon kills the mock daemon and cleans up temp files.
// Call this from TestMain after m.Run() returns.
func StopDaemon() {
    if daemon.proc != nil {
        daemon.proc.Kill()
        daemon.proc.Wait()
    }
    if daemon.tmpDir != "" {
        os.RemoveAll(daemon.tmpDir)
    }
    os.Unsetenv("DOCKER_HOST")
}

// TestEnv holds a fully wired test application with temp DB and mock Docker.
type TestEnv struct {
    App       *handlers.App
    Server    *httptest.Server
    WSServer  *ws.Server
    StacksDir string
    DataDir   string
    cancel    context.CancelFunc
}

// Setup creates a test environment with a real HTTP server, BoltDB, and mock Docker.
// Only uses "test-stack" from the daemon's stacks dir — fast for unit tests.
func Setup(t testing.TB) *TestEnv {
    return setupWithStacks(t, "test-stack")
}

// SetupFull creates a test environment with all 200+ stacks from test-data.
// Use only for stress tests and benchmarks.
func SetupFull(t testing.TB) *TestEnv {
    return setupWithStacks(t)
}

// setupWithStacks creates a test env. If the daemon was started via StartDaemon,
// uses its stacks dir. Otherwise falls back to copying stacks to a temp dir
// (for IDE test runners with DOCKER_HOST set externally).
func setupWithStacks(t testing.TB, stackNames ...string) *TestEnv {
    t.Helper()

    // Use daemon's stacks dir if available, otherwise fall back
    stacksDir := daemon.stacksDir
    if stacksDir == "" {
        stacksDir = fallbackStacksDir(t, stackNames...)
    } else if len(stackNames) > 0 {
        // Create a filtered view with symlinks so the App only sees named stacks
        filtered := t.TempDir()
        for _, name := range stackNames {
            src := filepath.Join(stacksDir, name)
            dst := filepath.Join(filtered, name)
            if err := os.Symlink(src, dst); err != nil {
                t.Fatal("symlink stack:", err)
            }
        }
        stacksDir = filtered
    }

    // Create temp data dir for BoltDB
    dataDir := filepath.Join(t.TempDir(), "data")

    // Open BoltDB in temp dir
    database, err := db.Open(dataDir)
    if err != nil {
        t.Fatal(err)
    }

    // Create stores
    users := models.NewUserStore(database)
    settings := models.NewSettingStore(database)
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

    // Connect to the mock daemon (or real Docker) via DOCKER_HOST
    dockerClient, err := docker.NewSDKClient()
    if err != nil {
        t.Fatal("new sdk client:", err)
    }

    // Force API version negotiation before any concurrent use. The Docker SDK
    // client with WithAPIVersionNegotiation() lazily writes the negotiated
    // version on the first request, which races if Events() and ContainerList()
    // fire concurrently from different goroutines. A ContainerList call triggers
    // the negotiation; subsequent calls are no-ops.
    if _, err := dockerClient.ContainerList(context.Background(), false, ""); err != nil {
        t.Fatal("pre-negotiate docker API version:", err)
    }

    // Terminal manager
    terms := terminal.NewManager()

    // WebSocket server
    wss := ws.NewServer()

    // Assemble App
    app := &handlers.App{
        Users:        users,
        Settings:     settings,
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
    handlers.RegisterStackHandlers(app)
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
    app.InitBroadcast()
    app.StartBroadcastWatcher(ctx)

    // Start test server
    server := httptest.NewServer(mux)

    t.Cleanup(func() {
        cancel()
        server.Close()
        dockerClient.Close()
        database.Close()
    })

    return &TestEnv{
        App:       app,
        Server:    server,
        WSServer:  wss,
        StacksDir: stacksDir,
        DataDir:   dataDir,
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

// SetStackRunning starts all containers for a stack via the Docker API.
// The test-stack's .mock.yaml sets services to "exited" by default; this
// calls POST /containers/{id}/start on each to transition them to "running".
func (e *TestEnv) SetStackRunning(t testing.TB, stackName string) {
    t.Helper()
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    containers, err := e.App.Docker.ContainerList(ctx, true, stackName)
    if err != nil {
        t.Fatal("list containers for SetStackRunning:", err)
    }
    for _, c := range containers {
        if err := e.App.Docker.ContainerStart(ctx, c.ID); err != nil {
            t.Fatalf("start container %s: %v", c.Name, err)
        }
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

// findStacksDirFatal locates test-data/stacks by walking up from cwd.
// Calls log.Fatalf on failure (for use in TestMain context where testing.TB
// is not available).
func findStacksDirFatal() string {
    dir, err := os.Getwd()
    if err != nil {
        log.Fatalf("testutil: cannot get cwd: %v", err)
    }
    for {
        candidate := filepath.Join(dir, "test-data", "stacks")
        if info, err := os.Stat(candidate); err == nil && info.IsDir() {
            abs, _ := filepath.Abs(candidate)
            return abs
        }
        parent := filepath.Dir(dir)
        if parent == dir {
            log.Fatalf("testutil: test-data/stacks not found")
        }
        dir = parent
    }
    panic("unreachable")
}

// fallbackStacksDir copies stacks from test-data into a temp dir for tests
// running without StartDaemon (e.g., IDE test runners with DOCKER_HOST set).
func fallbackStacksDir(t testing.TB, stackNames ...string) string {
    t.Helper()

    stacksSource := findTestdata(t)
    srcStacks := filepath.Join(stacksSource, "stacks")

    tmpDir := t.TempDir()
    stacksDir := filepath.Join(tmpDir, "stacks")
    if err := os.MkdirAll(stacksDir, 0755); err != nil {
        t.Fatal(err)
    }

    if len(stackNames) == 0 {
        copyDir(t, srcStacks, stacksDir)
    } else {
        for _, name := range stackNames {
            src := filepath.Join(srcStacks, name)
            dst := filepath.Join(stacksDir, name)
            if err := os.MkdirAll(dst, 0755); err != nil {
                t.Fatal(err)
            }
            copyDir(t, src, dst)
        }
    }
    return stacksDir
}

// findTestdata locates the test-data directory relative to the project root.
func findTestdata(t testing.TB) string {
    t.Helper()
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
    panic(fmt.Sprintf("unreachable"))
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
