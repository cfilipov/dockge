package main

import (
    "compress/gzip"
    "context"
    "fmt"
    "io"
    "io/fs"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "path/filepath"
    "strings"
    "syscall"
    "time"

    "github.com/cfilipov/dockge/backend-go/internal/compose"
    "github.com/cfilipov/dockge/backend-go/internal/config"
    "github.com/cfilipov/dockge/backend-go/internal/db"
    "github.com/cfilipov/dockge/backend-go/internal/docker"
    "github.com/cfilipov/dockge/backend-go/internal/handlers"
    "github.com/cfilipov/dockge/backend-go/internal/models"
    "github.com/cfilipov/dockge/backend-go/internal/terminal"
    "github.com/cfilipov/dockge/backend-go/internal/ws"
)

// version is set at build time via -ldflags="-X main.version=..."
var version = "1.5.0"

func main() {
    slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    })))

    cfg := config.Parse()

    slog.Info("starting dockge",
        "port", cfg.Port,
        "stacksDir", cfg.StacksDir,
        "dataDir", cfg.DataDir,
        "dev", cfg.Dev,
        "mock", cfg.Mock,
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

    // Frontend SPA handler
    var frontendFS fs.FS
    if cfg.Dev {
        // Serve from filesystem (for Vite HMR, point Vite proxy at this port)
        distPath := filepath.Join("..", "frontend-dist")
        slog.Info("dev mode: serving frontend from filesystem", "path", distPath)
        frontendFS = os.DirFS(distPath)
    } else {
        // Serve from embedded files
        sub, err := fs.Sub(frontendFiles, "frontend-dist")
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

    // Docker client (SDK or mock)
    dockerClient, err := docker.NewClient(cfg.Mock, cfg.StacksDir)
    if err != nil {
        slog.Error("docker client", "err", err)
        os.Exit(1)
    }
    defer dockerClient.Close()

    // Compose executor
    var composeExec compose.Composer
    if cfg.Mock {
        composeExec = compose.NewMockCompose(cfg.StacksDir)
    } else {
        composeExec = &compose.Exec{StacksDir: cfg.StacksDir}
    }

    // Terminal manager
    terms := terminal.NewManager()

    // Image update cache
    imageUpdates := models.NewImageUpdateStore(database)

    // Wire up handlers
    app := &handlers.App{
        Users:        users,
        Settings:     settings,
        Agents:       agents,
        ImageUpdates: imageUpdates,
        WS:           wss,
        Docker:       dockerClient,
        Compose:      composeExec,
        Terms:        terms,
        JWTSecret:    jwtSecret,
        NeedSetup:    userCount == 0,
        Version:      version,
        StacksDir:    cfg.StacksDir,
        Mock:         cfg.Mock,
    }
    handlers.RegisterAuthHandlers(app)
    handlers.RegisterSettingsHandlers(app)
    handlers.RegisterAgentHandlers(app)
    handlers.RegisterStackHandlers(app)
    handlers.RegisterTerminalHandlers(app)
    handlers.RegisterDockerHandlers(app)
    handlers.RegisterServiceHandlers(app)

    // Clean up terminal writers when a connection disconnects
    wss.OnDisconnect(func(c *ws.Conn) {
        terms.RemoveWriterFromAll(c.ID())
    })

    // Start background tasks
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    app.StartStackWatcher(ctx)
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

        gz, _ := gzip.NewWriterLevel(w, gzip.DefaultCompression)
        defer gz.Close()

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
