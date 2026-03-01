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

	// Dev mode: auto-seed admin user
	if cfg.Dev && userCount == 0 {
		if _, err := users.Create("admin", "testpass123"); err != nil {
			slog.Error("dev seed", "err", err)
		} else {
			slog.Info("dev mode: seeded admin user")
			userCount = 1
		}
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
	handlers.RegisterTerminalHandlers(app)
	handlers.RegisterDockerHandlers(app)
	handlers.RegisterServiceHandlers(app)

	// Dev mode: mock reset proxy endpoint.
	// Forwards POST /_mock/reset to the mock daemon over the DOCKER_HOST Unix socket,
	// then seeds BoltDB image updates from the response and triggers broadcasts.
	if cfg.Dev {
		mux.HandleFunc("POST /api/mock/reset", func(w http.ResponseWriter, _ *http.Request) {
			resp, err := resetViaDaemon()
			if err != nil {
				slog.Error("mock reset proxy", "err", err)
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}

			if resp.UpdateFlags != nil {
				if err := imageUpdates.SeedFromMock(resp.UpdateFlags); err != nil {
					slog.Error("seed image updates on mock reset", "err", err)
				}
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

// resetResponse is the JSON shape returned by the mock daemon's /_mock/reset.
type resetResponse struct {
	OK          bool            `json:"ok"`
	UpdateFlags map[string]bool `json:"updateFlags,omitempty"`
}

// resetViaDaemon sends POST /_mock/reset to the mock daemon over the DOCKER_HOST
// Unix socket and returns the parsed response. Returns an error if DOCKER_HOST
// is not a Unix socket (i.e., running against a real Docker daemon).
func resetViaDaemon() (*resetResponse, error) {
	dh := os.Getenv("DOCKER_HOST")
	if !strings.HasPrefix(dh, "unix://") {
		return nil, fmt.Errorf("DOCKER_HOST is not a Unix socket (got %q)", dh)
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
		return nil, fmt.Errorf("POST /_mock/reset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("/_mock/reset returned %d", resp.StatusCode)
	}

	var result resetResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode reset response: %w", err)
	}
	return &result, nil
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
