package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	netpprof "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cfilipov/dockge/internal/compose"
	"github.com/cfilipov/dockge/internal/config"
	"github.com/cfilipov/dockge/internal/db"
	dbgmem "github.com/cfilipov/dockge/internal/debug"
	"github.com/cfilipov/dockge/internal/docker"
	"github.com/cfilipov/dockge/internal/handlers"
	"github.com/cfilipov/dockge/internal/models"
	"github.com/cfilipov/dockge/internal/stack"
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

	// Cap GOMAXPROCS to reduce per-P memory overhead (mcache, sync.Pool shards,
	// gzip writers). Default 1 is plenty for a single-user web app. Set to 0
	// or use DOCKGE_MAX_PROCS=0 to keep the Go default (= host CPU count).
	if cfg.MaxProcs > 0 {
		runtime.GOMAXPROCS(cfg.MaxProcs)
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	})))

	slog.Info("starting dockge",
		"port", cfg.Port,
		"stacksDir", cfg.StacksDir,
		"dataDir", cfg.DataDir,
		"dev", cfg.Dev,
		"pprof", cfg.Dev || cfg.Pprof,
		"logLevel", cfg.LogLevel,
		"noAuth", cfg.NoAuth,
		"maxProcs", runtime.GOMAXPROCS(0),
	)

	// Open database
	database, err := db.Open(cfg.DataDir)
	if err != nil {
		slog.Error("database", "err", err)
		os.Exit(1)
	}
	defer database.Close()

	// WebSocket server
	wss := ws.NewServer(cfg.Dev)

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

		// In-process memory tracker — samples every 50ms without forced GC.
		// Stops when ctx is cancelled (graceful shutdown, declared below).
		// We create a child context here and cancel it on shutdown.
		memCtx, memCancel := context.WithCancel(context.Background())
		tracker := dbgmem.NewMemTracker(memCtx, 50*time.Millisecond)
		defer memCancel()
		mux.HandleFunc("GET /api/debug/memstats", tracker.HandleGet)
		mux.HandleFunc("POST /api/debug/memstats/reset", tracker.HandleReset)
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

	// Dev mode: auto-create admin user if no users exist
	if cfg.Dev && userCount == 0 {
		if _, err := users.Create("admin", "testpass123"); err != nil {
			slog.Error("dev auto-create admin", "err", err)
			os.Exit(1)
		}
		slog.Info("dev mode: created admin user", "username", "admin")
		userCount = 1
	}

	// Docker client — connects to whatever DOCKER_HOST points to.
	// In dev+mock environments, the external mock-daemon sets DOCKER_HOST
	// to its Unix socket. In production, it connects to the real Docker daemon.
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

	// Wire up handlers
	app := &handlers.App{
		Users:        users,
		Settings:     settings,
		ImageUpdates: imageUpdates,
		WS:           wss,
		Docker:       dockerClient,
		Terms:        terms,
		StackLocks:   stack.NewNamedMutex(),
		LoginLimiter: handlers.NewLoginRateLimiter(5, 15*time.Minute),
		JWTSecret:    jwtSecret,
		NeedSetup:    userCount == 0,
		Version:      version,
		StacksDir:    cfg.StacksDir,
		NoAuth:       cfg.NoAuth,
		Dev:          cfg.Dev,
	}
	handlers.RegisterAuthHandlers(app)
	handlers.RegisterSettingsHandlers(app)
	handlers.RegisterStackHandlers(app)
	handlers.RegisterDockerHandlers(app)
	handlers.RegisterServiceHandlers(app)
	handlers.RegisterTerminalHandlers(app)

	// Dev mode: broadcast metrics and mock reset proxy endpoints.
	if cfg.Dev {
		mux.HandleFunc("GET /api/broadcast-metrics", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(app.BcastMetrics.Snapshot())
		})
	}

	// Dev mode: mock reset proxy endpoint.
	// Forwards POST /_mock/reset to the mock daemon over the DOCKER_HOST Unix socket,
	// then triggers broadcasts so the frontend sees fresh state.
	if cfg.Dev {
		mux.HandleFunc("POST /api/mock/reset", func(w http.ResponseWriter, _ *http.Request) {
			if err := resetViaDaemon(); err != nil {
				slog.Error("mock reset proxy", "err", err)
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}

			// Mock reset bypasses Docker commands, so no events fire.
			// Trigger all broadcasts explicitly.
			app.TriggerStacksBroadcast()
			app.TriggerContainersBroadcast()
			app.TriggerNetworksBroadcast()
			app.TriggerImagesBroadcast()
			app.TriggerVolumesBroadcast()
			app.TriggerUpdatesBroadcast()
			slog.Info("mock state reset via daemon")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})

		// Dev mode: reset DB state (users, rate limiter) to pristine dev defaults.
		// Separate from mock/reset which only resets Docker state.
		mux.HandleFunc("POST /api/dev/reset-db", func(w http.ResponseWriter, _ *http.Request) {
			// Wipe all users and re-create admin with default password
			if err := users.DeleteAll(); err != nil {
				slog.Error("dev reset-db: delete users", "err", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if _, err := users.Create("admin", "testpass123"); err != nil {
				slog.Error("dev reset-db: create admin", "err", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			app.NeedSetup = false

			// Clear rate limiter state
			if app.LoginLimiter != nil {
				app.LoginLimiter.ResetAll()
			}

			slog.Info("dev DB state reset")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})
	}

	// No-auth mode: auto-authenticate every connection as user 1
	// and send initial state (afterLogin starts the watcher + sends all 6 channels).
	// NOTE: This overrides the HandleConnect set in RegisterAuth, so we must
	// also send the "info" event that the auth handler's connect callback sends.
	if cfg.NoAuth {
		slog.Warn("authentication disabled (--no-auth)")
		wss.HandleConnect(func(c *ws.Conn) {
			// Send server info (normally sent by auth's connect handler)
			ws.SendEvent(c, "info", map[string]interface{}{
				"version":       app.Version,
				"latestVersion": app.Version,
				"isContainer":   true,
				"dev":           app.Dev,
			})
			c.SetUser(1)
			app.AfterLogin(c)
		})
	}

	// Clean up terminal writers and stats subscriptions when a connection disconnects.
	wss.OnDisconnect(func(c *ws.Conn) {
		// Drain all terminal sessions and clean up each one
		for _, s := range c.DrainSessions() {
			terms.RemoveWriterAndCleanup(s.TermName, s.WriterKey)
		}
		app.CancelStatsSub(c.ID())
		app.CancelTopSub(c.ID())
	})

	// Start background tasks
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set GOMEMLIMIT to cap heap size unless the user overrides via env.
	if os.Getenv("GOMEMLIMIT") == "" {
		debug.SetMemoryLimit(32 << 20) // 32 MiB
	}

	// Initialize broadcast infrastructure (debouncer, event bus, hash state)
	app.InitBroadcast()

	// Start periodic terminal cleanup (removes completed terminals with no writers)
	terms.StartCleanupLoop(ctx)

	// Start compose file watcher (fsnotify) — triggers broadcast on file changes
	if err := compose.StartWatcher(ctx, cfg.StacksDir, func(stackName string) {
		app.TriggerStacksBroadcast()
	}); err != nil {
		slog.Warn("compose file watcher failed to start", "err", err)
	}

	// Start the broadcast watcher at boot — it runs forever. Individual
	// broadcast functions skip Docker API calls when no clients are connected.
	app.StartBroadcastWatcher(ctx)
	app.StartImageUpdateChecker(ctx)

	// Periodically return unused memory to the OS. Go's runtime retains
	// freed heap pages as RSS for future allocations; this nudges it to
	// release them sooner, keeping steady-state RSS lower.
	// When no clients are connected, also close idle Docker HTTP connections
	// and log memory diagnostics.
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !wss.HasAuthenticatedConns() {
					dockerClient.CloseIdleConnections()
					var m runtime.MemStats
					runtime.ReadMemStats(&m)
					slog.Debug("idle memory stats",
						"heapAlloc", formatBytes(m.HeapAlloc),
						"heapInuse", formatBytes(m.HeapInuse),
						"heapIdle", formatBytes(m.HeapIdle),
						"heapReleased", formatBytes(m.HeapReleased),
						"sys", formatBytes(m.Sys),
					)
				}
				debug.FreeOSMemory()
			}
		}
	}()

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

// resetViaDaemon sends POST /_mock/reset to the mock daemon over the DOCKER_HOST
// Unix socket. Returns an error if DOCKER_HOST is not a Unix socket (i.e.,
// running against a real Docker daemon).
func resetViaDaemon() error {
	dh := os.Getenv("DOCKER_HOST")
	if !strings.HasPrefix(dh, "unix://") {
		return fmt.Errorf("DOCKER_HOST is not a Unix socket (got %q)", dh)
	}
	sockPath := strings.TrimPrefix(dh, "unix://")

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.DialTimeout("unix", sockPath, 2*time.Second)
			},
		},
		Timeout: 10 * time.Second,
	}

	resp, err := client.Post("http://docker/_mock/reset", "", nil)
	if err != nil {
		return fmt.Errorf("POST /_mock/reset: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("/_mock/reset returned %d", resp.StatusCode)
	}
	return nil
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

// formatBytes formats a byte count as a human-readable string for log output.
func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

