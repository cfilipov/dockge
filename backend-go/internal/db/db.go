package db

import (
    "database/sql"
    "embed"
    "fmt"
    "log/slog"
    "os"
    "path/filepath"

    _ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func Open(dataDir string) (*sql.DB, error) {
    if err := os.MkdirAll(dataDir, 0755); err != nil {
        return nil, fmt.Errorf("create data dir: %w", err)
    }

    dbPath := filepath.Join(dataDir, "dockge.db")
    db, err := sql.Open("sqlite", dbPath)
    if err != nil {
        return nil, fmt.Errorf("open sqlite: %w", err)
    }

    // Single connection for SQLite (WAL mode handles concurrency)
    db.SetMaxOpenConns(1)

    if err := setPragmas(db); err != nil {
        db.Close()
        return nil, fmt.Errorf("set pragmas: %w", err)
    }

    if err := runMigrations(db); err != nil {
        db.Close()
        return nil, fmt.Errorf("run migrations: %w", err)
    }

    slog.Info("database ready", "path", dbPath)
    return db, nil
}

func setPragmas(db *sql.DB) error {
    pragmas := []string{
        "PRAGMA journal_mode = WAL",
        "PRAGMA synchronous = NORMAL",
        "PRAGMA foreign_keys = ON",
        "PRAGMA busy_timeout = 5000",
    }
    for _, p := range pragmas {
        if _, err := db.Exec(p); err != nil {
            return fmt.Errorf("%s: %w", p, err)
        }
    }
    return nil
}

func runMigrations(db *sql.DB) error {
    // Create migrations tracking table
    _, err := db.Exec(`CREATE TABLE IF NOT EXISTS migrations (
        id      INTEGER PRIMARY KEY,
        name    TEXT UNIQUE NOT NULL,
        applied INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
    )`)
    if err != nil {
        return fmt.Errorf("create migrations table: %w", err)
    }

    entries, err := migrationsFS.ReadDir("migrations")
    if err != nil {
        return fmt.Errorf("read migrations dir: %w", err)
    }

    for _, entry := range entries {
        name := entry.Name()

        // Check if already applied
        var count int
        if err := db.QueryRow("SELECT COUNT(*) FROM migrations WHERE name = ?", name).Scan(&count); err != nil {
            return fmt.Errorf("check migration %s: %w", name, err)
        }
        if count > 0 {
            continue
        }

        // Read and execute
        content, err := migrationsFS.ReadFile(filepath.Join("migrations", name))
        if err != nil {
            return fmt.Errorf("read migration %s: %w", name, err)
        }

        if _, err := db.Exec(string(content)); err != nil {
            return fmt.Errorf("execute migration %s: %w", name, err)
        }

        if _, err := db.Exec("INSERT INTO migrations (name) VALUES (?)", name); err != nil {
            return fmt.Errorf("record migration %s: %w", name, err)
        }

        slog.Info("applied migration", "name", name)
    }

    return nil
}
