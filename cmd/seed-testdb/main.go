// Command seed-testdb initializes a BoltDB database for dev/test use.
// It creates the admin user, generates a JWT secret, seeds image update
// flags from mock.yaml files, and stamps imageUpdateLastCheck to prevent
// the background checker from running immediately on startup.
//
// Usage:
//
//	seed-testdb --data-dir test-data --stacks-dir test-data/stacks
package main

import (
	"flag"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/cfilipov/dockge/internal/db"
	"github.com/cfilipov/dockge/internal/docker/mock"
	"github.com/cfilipov/dockge/internal/models"
)

func main() {
	var (
		dataDir   string
		stacksDir string
		username  string
		password  string
		logLevel  string
	)

	flag.StringVar(&dataDir, "data-dir", "test-data", "Path to data directory (BoltDB)")
	flag.StringVar(&stacksDir, "stacks-dir", "test-data/stacks", "Path to stacks directory")
	flag.StringVar(&username, "username", "admin", "Admin username to create")
	flag.StringVar(&password, "password", "testpass123", "Admin password")
	flag.StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: parseLogLevel(logLevel),
	})))

	// Open database
	database, err := db.Open(dataDir)
	if err != nil {
		slog.Error("open database", "err", err)
		os.Exit(1)
	}
	defer database.Close()

	// Create admin user (skip if users already exist)
	users := models.NewUserStore(database)
	count, err := users.Count()
	if err != nil {
		slog.Error("count users", "err", err)
		os.Exit(1)
	}
	if count == 0 {
		if _, err := users.Create(username, password); err != nil {
			slog.Error("create admin user", "err", err)
			os.Exit(1)
		}
		slog.Info("created admin user", "username", username)
	} else {
		slog.Info("users already exist, skipping admin creation", "count", count)
	}

	// Generate JWT secret
	settings := models.NewSettingStore(database)
	if _, err := settings.EnsureJWTSecret(); err != nil {
		slog.Error("ensure JWT secret", "err", err)
		os.Exit(1)
	}

	// Seed image update flags from mock.yaml files
	mockData := mock.BuildMockData(stacksDir)
	imageUpdates := models.NewImageUpdateStore(database)
	flags := mockData.UpdateFlags()
	if len(flags) > 0 {
		if err := imageUpdates.SeedFromMock(flags); err != nil {
			slog.Error("seed image updates", "err", err)
			os.Exit(1)
		}
		slog.Info("seeded image update flags", "count", len(flags))
	}

	// Stamp last check time to prevent background checker from running immediately
	if err := imageUpdates.SetLastCheckTime(time.Now()); err != nil {
		slog.Error("set last check time", "err", err)
		os.Exit(1)
	}

	slog.Info("test database seeded successfully", "dataDir", dataDir)
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
