package config

import (
    "flag"
    "log/slog"
    "os"
    "strconv"
    "strings"
)

type Config struct {
    Port      int
    StacksDir string
    DataDir   string
    Dev       bool
    LogLevel  slog.Level // Parsed log level (debug, info, warn, error)
    NoAuth    bool       // Skip authentication (all endpoints open)
    Pprof     bool       // Enable /debug/pprof/ endpoints
}

func Parse() *Config {
    cfg := &Config{}

    var logLevel string
    flag.IntVar(&cfg.Port, "port", 5001, "HTTP server port")
    flag.StringVar(&cfg.StacksDir, "stacks-dir", "/opt/stacks", "Path to stacks directory")
    flag.StringVar(&cfg.DataDir, "data-dir", "./data", "Path to data directory (SQLite DB)")
    flag.BoolVar(&cfg.Dev, "dev", false, "Development mode (serve frontend from filesystem)")
    flag.StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
    flag.BoolVar(&cfg.NoAuth, "no-auth", false, "Disable authentication (all endpoints open)")
    flag.Parse()

    // Env vars override flags (if set)
    if v := os.Getenv("DOCKGE_PORT"); v != "" {
        if p, err := strconv.Atoi(v); err == nil {
            cfg.Port = p
        }
    }
    if v := os.Getenv("DOCKGE_STACKS_DIR"); v != "" {
        cfg.StacksDir = v
    }
    if v := os.Getenv("DOCKGE_DATA_DIR"); v != "" {
        cfg.DataDir = v
    }
    if v := os.Getenv("DOCKGE_LOG_LEVEL"); v != "" {
        logLevel = v
    }
    if v := os.Getenv("DOCKGE_NO_AUTH"); v == "1" || v == "true" {
        cfg.NoAuth = true
    }
    if v := os.Getenv("DOCKGE_PPROF"); v == "1" || v == "true" {
        cfg.Pprof = true
    }

    cfg.LogLevel = parseLogLevel(logLevel)

    return cfg
}

func parseLogLevel(s string) slog.Level {
    switch strings.ToLower(strings.TrimSpace(s)) {
    case "debug":
        return slog.LevelDebug
    case "warn", "warning":
        return slog.LevelWarn
    case "error":
        return slog.LevelError
    default:
        return slog.LevelInfo
    }
}
