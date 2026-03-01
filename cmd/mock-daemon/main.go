// Command mock-daemon runs a standalone fake Docker daemon on a Unix socket.
// It absorbs all mock startup logic that used to live in main.go behind --mock,
// making the main dockge binary completely unaware of mock mode.
//
// Usage:
//
//	mock-daemon --socket /tmp/dockge-mock/docker.sock \
//	            --stacks-source test-data/stacks \
//	            --stacks-dir test-data/stacks
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/cfilipov/dockge/internal/docker/mock"
)

func main() {
	var (
		socketPath   string
		stacksSource string
		stacksDir    string
		logLevel     string
	)

	flag.StringVar(&socketPath, "socket", "", "Unix socket path (default: /tmp/dockge-mock-<pid>/docker.sock)")
	flag.StringVar(&stacksSource, "stacks-source", "test-data/stacks", "Pristine source stacks directory")
	flag.StringVar(&stacksDir, "stacks-dir", "test-data/stacks", "Working stacks directory that dockge reads from")
	flag.StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: parseLogLevel(logLevel),
	})))

	// Default socket path if not specified
	if socketPath == "" {
		dir := fmt.Sprintf("/tmp/dockge-mock-%d", os.Getpid())
		if err := os.MkdirAll(dir, 0755); err != nil {
			slog.Error("create socket dir", "err", err)
			os.Exit(1)
		}
		socketPath = dir + "/docker.sock"
	}

	// Build mock data from stacks directory
	mockData := mock.BuildMockData(stacksDir)
	mockState := mock.DefaultDevStateFromData(mockData)

	// Start fake daemon on the specified socket
	cleanup, err := mock.StartFakeDaemonOnSocket(mockState, mockData, stacksDir, stacksSource, socketPath)
	if err != nil {
		slog.Error("start fake daemon", "err", err)
		os.Exit(1)
	}
	defer cleanup()

	// Print socket path to stdout so parent processes can discover it
	fmt.Println(socketPath)

	slog.Info("mock daemon started",
		"socket", socketPath,
		"stacksDir", stacksDir,
		"stacksSource", stacksSource,
	)

	// Wait for SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("mock daemon shutting down")
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
