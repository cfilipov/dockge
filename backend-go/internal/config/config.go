package config

import (
    "flag"
    "os"
    "strconv"
)

type Config struct {
    Port      int
    StacksDir string
    DataDir   string
    Dev       bool
}

func Parse() *Config {
    cfg := &Config{}

    flag.IntVar(&cfg.Port, "port", 5001, "HTTP server port")
    flag.StringVar(&cfg.StacksDir, "stacks-dir", "/opt/stacks", "Path to stacks directory")
    flag.StringVar(&cfg.DataDir, "data-dir", "./data", "Path to data directory (SQLite DB)")
    flag.BoolVar(&cfg.Dev, "dev", false, "Development mode (serve frontend from filesystem)")
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

    return cfg
}
