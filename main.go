package main

import (
    "compress/gzip"
    "context"
    "fmt"
    "io"
    "io/fs"
    "log/slog"
    "net/http"
    netpprof "net/http/pprof"
    "os"
    "os/signal"
    "path/filepath"
    "strings"
    "sync"
    "syscall"
    "time"

    "github.com/cfilipov/dockge/internal/compose"
    "github.com/cfilipov/dockge/internal/config"
    "github.com/cfilipov/dockge/internal/db"
    "github.com/cfilipov/dockge/internal/docker"
    "github.com/cfilipov/dockge/internal/handlers"
    "github.com/cfilipov/dockge/internal/models"
    "github.com/cfilipov/dockge/internal/terminal"
    "github.com/cfilipov/dockge/internal/ws"
)

// version is set at build time via -ldflags="-X main.version=..."
var version = "1.5.0"

func main() {
    // Quick healthcheck mode — used by Docker HEALTHCHECK from scratch image.
    // Avoids needing wget/curl in the container. The binary starts in ~10ms,
    // hits /healthz, and exits immediately — no server initialization.
    if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
        port := "5001"
        if v := os.Getenv("DOCKGE_PORT"); v != "" {
            port = v
        }
        resp, err := http.Get("http://127.0.0.1:" + port + "/healthz")
        if err != nil || resp.StatusCode != 200 {
            os.Exit(1)
        }
        os.Exit(0)
    }

    cfg := config.Parse()

    slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
        Level: cfg.LogLevel,
    })))

    slog.Info("starting dockge",
        "port", cfg.Port,
        "stacksDir", cfg.StacksDir,
        "dataDir", cfg.DataDir,
        "dev", cfg.Dev,
        "mock", cfg.Mock,
        "pprof", cfg.Dev || cfg.Pprof,
        "logLevel", cfg.LogLevel,
        "noAuth", cfg.NoAuth,
    )

    // Open database
    database, err := db.Open(cfg.DataDir)
    if err != nil {
        slog.Error("database", "err", err)
        os.Exit(1)
    }
    defer database.Close()

    // WebSocket server
    wss := ws.NewServer()

    // HTTP mux
    mux := http.NewServeMux()
    mux.Handle("/ws", wss.UpgradeHandler())
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("ok"))
    })

    // Enable pprof endpoints in dev mode or via DOCKGE_PPROF=1
    if cfg.Dev || cfg.Pprof {
        mux.HandleFunc("/debug/pprof/", pprofIndex)
        mux.HandleFunc("/debug/pprof/cmdline", pprofCmdline)
        mux.HandleFunc("/debug/pprof/profile", pprofProfile)
        mux.HandleFunc("/debug/pprof/symbol", pprofSymbol)
        mux.HandleFunc("/debug/pprof/trace", pprofTrace)
        slog.Info("pprof enabled at /debug/pprof/")
    }

    // Frontend SPA handler
    var frontendFS fs.FS
    if cfg.Dev {
        // Serve from filesystem (for Vite HMR, point Vite proxy at this port)
        distPath := "dist"
        slog.Info("dev mode: serving frontend from filesystem", "path", distPath)
        frontendFS = os.DirFS(distPath)
    } else {
        // Serve from embedded files
        sub, err := fs.Sub(staticFiles, "dist")
        if err != nil {
            slog.Error("embed frontend", "err", err)
            os.Exit(1)
        }
        frontendFS = sub
    }
    mux.Handle("/", gzipMiddleware(spaHandler(frontendFS)))

    // Models
    users := models.NewUserStore(database)
    settings := models.NewSettingStore(database)
    agents := models.NewAgentStore(database)

    // JWT secret (auto-generated on first run)
    jwtSecret, err := settings.EnsureJWTSecret()
    if err != nil {
        slog.Error("jwt secret", "err", err)
        os.Exit(1)
    }

    // Check if setup is needed
    userCount, err := users.Count()
    if err != nil {
        slog.Error("user count", "err", err)
        os.Exit(1)
    }

    // Dev mode: auto-seed admin user and test stacks when mock is enabled
    if cfg.Dev && cfg.Mock {
        if userCount == 0 {
            if _, err := users.Create("admin", "testpass123"); err != nil {
                slog.Error("dev seed", "err", err)
            } else {
                slog.Info("dev mode: seeded admin user")
                userCount = 1
            }
        }
        seedDevStacks(cfg.StacksDir)
    }

    // Mock state, data, and fake daemon
    var mockState *docker.MockState
    var mockData *docker.MockData
    if cfg.Mock {
        mockData = docker.BuildMockData(cfg.StacksDir)
        mockState = docker.DefaultDevStateFromData(mockData)

        // Start fake Docker daemon on a Unix socket
        sockPath, daemonCleanup, err := docker.StartFakeDaemon(mockState, mockData, cfg.StacksDir)
        if err != nil {
            slog.Error("fake docker daemon", "err", err)
            os.Exit(1)
        }
        defer daemonCleanup()

        // Set DOCKER_HOST so SDKClient and mock binary both connect to fake daemon
        os.Setenv("DOCKER_HOST", "unix://"+sockPath)
        slog.Info("mock mode: fake daemon started", "socket", sockPath)

        // Put mock docker binary first on PATH so exec.Command("docker", ...) resolves to it
        mockBinDir, _ := filepath.Abs("test-data/bin")
        os.Setenv("PATH", mockBinDir+":"+os.Getenv("PATH"))
        slog.Info("mock mode: mock docker binary on PATH", "dir", mockBinDir)
    }

    // Docker client — always uses SDKClient. In mock mode, DOCKER_HOST points
    // to the fake daemon. In real mode, it connects to the real Docker daemon.
    dockerClient, err := docker.NewSDKClient()
    if err != nil {
        slog.Error("docker client", "err", err)
        os.Exit(1)
    }
    defer dockerClient.Close()

    // Terminal manager
    terms := terminal.NewManager()

    // Image update cache
    imageUpdates := models.NewImageUpdateStore(database)

    // In mock mode, seed BoltDB image updates from mock.yaml flags
    if cfg.Mock && mockData != nil {
        if err := imageUpdates.SeedFromMock(mockData.UpdateFlags()); err != nil {
            slog.Error("seed image updates from mock", "err", err)
        } else {
            slog.Info("mock mode: seeded image update flags", "count", len(mockData.UpdateFlags()))
        }
    }

    // Wire up handlers
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
        Version:      version,
        StacksDir:    cfg.StacksDir,
        NoAuth:       cfg.NoAuth,
    }
    handlers.RegisterAuthHandlers(app)
    handlers.RegisterSettingsHandlers(app)
    handlers.RegisterAgentHandlers(app)
    handlers.RegisterStackHandlers(app)
    handlers.RegisterTerminalHandlers(app)
    handlers.RegisterDockerHandlers(app)
    handlers.RegisterServiceHandlers(app)

    // Mock state reset endpoint (test use only).
    if cfg.Mock && mockState != nil {
        // Source stacks directory: same parent as stacksDir but named "stacks"
        // (matches the convention in playwright.config.ts: cp -a test-data/stacks test-data/e2e-stacks)
        stacksSource := filepath.Join(filepath.Dir(cfg.StacksDir), "stacks")

        mux.HandleFunc("POST /api/mock/reset", func(w http.ResponseWriter, _ *http.Request) {
            mockState.Reset()
            if mockData != nil {
                if err := imageUpdates.SeedFromMock(mockData.UpdateFlags()); err != nil {
                    slog.Error("seed image updates on mock reset", "err", err)
                }
            }

            // Restore stacks directory from source to undo any filesystem
            // changes made by tests (deploy creating dirs, delete removing dirs).
            // Clear contents instead of removing the directory itself so the
            // fsnotify watcher (which watches this inode) stays valid.
            if info, err := os.Stat(stacksSource); err == nil && info.IsDir() {
                if err := clearDirContents(cfg.StacksDir); err != nil {
                    slog.Error("mock reset: clear stacks dir", "err", err)
                }
                if err := copyDirRecursive(stacksSource, cfg.StacksDir); err != nil {
                    slog.Error("mock reset: copy stacks dir", "err", err)
                }
            }

            // Clear any test-created agents from BoltDB
            if err := app.Agents.ClearAll(); err != nil {
                slog.Error("mock reset: clear agents", "err", err)
            }

            // Mock reset bypasses Docker commands, so no events fire.
            // Trigger all broadcasts explicitly.
            app.BroadcastAll()
            app.TriggerStacksBroadcast()
            app.TriggerContainersBroadcast()
            app.TriggerNetworksBroadcast()
            app.TriggerImagesBroadcast()
            app.TriggerVolumesBroadcast()
            app.TriggerUpdatesBroadcast()
            slog.Info("mock state reset to default")
            w.WriteHeader(http.StatusOK)
            w.Write([]byte("ok"))
        })
    }

    // No-auth mode: auto-authenticate every connection as user 1
    if cfg.NoAuth {
        slog.Warn("authentication disabled (--no-auth)")
        wss.HandleConnect(func(c *ws.Conn) {
            c.SetUser(1)
        })
    }

    // Clean up terminal writers when a connection disconnects
    wss.OnDisconnect(func(c *ws.Conn) {
        terms.RemoveWriterFromAll(c.ID())
    })

    // Start background tasks
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Initialize broadcast infrastructure
    app.InitBroadcast()

    // Start compose file watcher (fsnotify) — triggers broadcast on file changes
    if err := compose.StartWatcher(ctx, cfg.StacksDir, func(stackName string) {
        app.TriggerStacksBroadcast()
    }); err != nil {
        slog.Warn("compose file watcher failed to start", "err", err)
    }

    app.StartStackWatcher(ctx)
    app.StartBroadcastWatcher(ctx)
    app.StartImageUpdateChecker(ctx)

    // Start HTTP server
    addr := fmt.Sprintf(":%d", cfg.Port)
    srv := &http.Server{
        Addr:         addr,
        Handler:      mux,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    go func() {
        slog.Info("listening", "addr", addr)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            slog.Error("server", "err", err)
            os.Exit(1)
        }
    }()

    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    slog.Info("shutting down")
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer shutdownCancel()
    srv.Shutdown(shutdownCtx)
}

// spaHandler serves static files from the given FS. If the requested file
// doesn't exist, it falls back to index.html for client-side routing.
func spaHandler(fsys fs.FS) http.Handler {
    fileServer := http.FileServer(http.FS(fsys))
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Clean the path
        path := strings.TrimPrefix(r.URL.Path, "/")
        if path == "" {
            path = "index.html"
        }

        // Try to open the file
        f, err := fsys.Open(path)
        if err != nil {
            // File not found — serve index.html for SPA routing
            r.URL.Path = "/"
            fileServer.ServeHTTP(w, r)
            return
        }
        f.Close()

        // File exists — serve it
        fileServer.ServeHTTP(w, r)
    })
}

// pprof handler wrappers — net/http/pprof registers on DefaultServeMux via init(),
// but we use a custom mux. Reference the exported handler functions directly.
var (
    pprofIndex   = netpprof.Index
    pprofCmdline = netpprof.Cmdline
    pprofProfile = netpprof.Profile
    pprofSymbol  = netpprof.Symbol
    pprofTrace   = netpprof.Trace
)

// gzipPool reuses gzip.Writer instances (~256KB internal state each).
var gzipPool = sync.Pool{
    New: func() any {
        w, _ := gzip.NewWriterLevel(nil, gzip.DefaultCompression)
        return w
    },
}

// gzipMiddleware compresses responses on the fly for clients that accept it.
func gzipMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
            next.ServeHTTP(w, r)
            return
        }

        // Skip compression for small/binary responses
        path := r.URL.Path
        ext := filepath.Ext(path)
        switch ext {
        case ".png", ".jpg", ".jpeg", ".gif", ".ico", ".woff", ".woff2", ".br", ".gz":
            next.ServeHTTP(w, r)
            return
        }

        gz := gzipPool.Get().(*gzip.Writer)
        gz.Reset(w)
        defer func() {
            gz.Close()
            gzipPool.Put(gz)
        }()

        w.Header().Set("Content-Encoding", "gzip")
        w.Header().Del("Content-Length")

        next.ServeHTTP(&gzipResponseWriter{Writer: gz, ResponseWriter: w}, r)
    })
}

type gzipResponseWriter struct {
    io.Writer
    http.ResponseWriter
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
    return w.Writer.Write(b)
}

// seedDevStacks copies test-data stacks into the stacks directory for dev+mock mode.
// It uses a marker file (.dockge-dev-stacks) to avoid re-copying on subsequent runs.
// If the directory has user content (no marker, not empty), it's left alone.
func seedDevStacks(stacksDir string) {
    marker := filepath.Join(stacksDir, ".dockge-dev-stacks")

    // If marker exists, already seeded
    if _, err := os.Stat(marker); err == nil {
        slog.Debug("dev stacks already seeded")
        return
    }

    // If directory has content but no marker, it's user data — don't overwrite
    entries, err := os.ReadDir(stacksDir)
    if err == nil && len(entries) > 0 {
        slog.Info("stacks dir has existing content, skipping dev seed")
        return
    }

    // Find test-data/stacks/ relative to cwd
    srcDir := filepath.Join("test-data", "stacks")
    if info, err := os.Stat(srcDir); err != nil || !info.IsDir() {
        slog.Warn("test-data/stacks not found, skipping dev stack seed")
        return
    }

    // Copy all stacks
    count := 0
    srcEntries, err := os.ReadDir(srcDir)
    if err != nil {
        slog.Warn("read test-data/stacks", "err", err)
        return
    }

    for _, entry := range srcEntries {
        if !entry.IsDir() {
            continue
        }
        dst := filepath.Join(stacksDir, entry.Name())
        if err := os.MkdirAll(dst, 0755); err != nil {
            slog.Warn("mkdir", "path", dst, "err", err)
            continue
        }
        if err := copyDirRecursive(filepath.Join(srcDir, entry.Name()), dst); err != nil {
            slog.Warn("copy stack", "name", entry.Name(), "err", err)
            continue
        }
        count++
    }

    // Write marker
    os.WriteFile(marker, []byte("seeded by dockge dev mode\n"), 0644)
    slog.Info("dev mode: seeded test stacks", "count", count)
}

// clearDirContents removes all entries inside dir without removing dir itself.
// This preserves the directory inode so fsnotify watchers remain valid.
func clearDirContents(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

// copyDirRecursive copies all files from src to dst recursively.
func copyDirRecursive(src, dst string) error {
    return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        rel, err := filepath.Rel(src, path)
        if err != nil {
            return err
        }
        target := filepath.Join(dst, rel)
        if d.IsDir() {
            return os.MkdirAll(target, 0755)
        }
        data, err := os.ReadFile(path)
        if err != nil {
            return err
        }
        return os.WriteFile(target, data, 0644)
    })
}
