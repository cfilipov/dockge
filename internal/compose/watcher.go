package compose

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// StartWatcher watches the stacks directory tree for compose file changes.
// On change, calls onChange(stackName) so the caller can broadcast fresh data.
// On error it retries with exponential backoff; after repeated failures it
// exits the process.
func StartWatcher(ctx context.Context, stacksDir string, onChange func(stackName string)) error {
	// Verify the directory exists before starting
	if _, err := os.Stat(stacksDir); err != nil {
		return err
	}

	go runWatcherLoop(ctx, stacksDir, onChange)
	return nil
}

// runWatcherLoop creates an fsnotify watcher and processes events.
// On error or channel close, it retries with exponential backoff up to
// maxRetries times, then exits the process.
func runWatcherLoop(ctx context.Context, stacksDir string, onChange func(stackName string)) {
	const maxRetries = 5
	failures := 0
	backoff := 1 * time.Second

	for {
		err := runWatcher(ctx, stacksDir, onChange)
		if ctx.Err() != nil {
			return // clean shutdown
		}

		failures++
		if failures > maxRetries {
			slog.Error("compose file watcher: too many failures, exiting", "failures", failures, "lastErr", err)
			os.Exit(1)
		}

		slog.Warn("compose file watcher: retrying", "attempt", failures, "backoff", backoff, "err", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, 30*time.Second)
	}
}

// runWatcher creates an fsnotify watcher, processes events until an error
// occurs or a channel closes, then returns the error.
func runWatcher(ctx context.Context, stacksDir string, onChange func(stackName string)) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	defer watcher.Close()

	// Watch the top-level stacks directory (for new/removed stack subdirs)
	if err := watcher.Add(stacksDir); err != nil {
		return fmt.Errorf("watch stacks dir: %w", err)
	}

	// Watch each existing stack subdirectory
	entries, err := os.ReadDir(stacksDir)
	if err != nil {
		return fmt.Errorf("read stacks dir: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			subdir := filepath.Join(stacksDir, entry.Name())
			if err := watcher.Add(subdir); err != nil {
				slog.Warn("compose watcher: add subdir", "err", err, "dir", subdir)
			}
		}
	}

	slog.Info("compose file watcher started", "dir", stacksDir)

	// Debounce: coalesce events for the same stack within 200ms
	var debounceMu sync.Mutex
	pending := make(map[string]*time.Timer)

	triggerUpdate := func(stackName string) {
		debounceMu.Lock()
		defer debounceMu.Unlock()

		if timer, ok := pending[stackName]; ok {
			timer.Stop()
		}
		pending[stackName] = time.AfterFunc(200*time.Millisecond, func() {
			debounceMu.Lock()
			delete(pending, stackName)
			debounceMu.Unlock()

			slog.Debug("compose watcher: file changed", "stack", stackName)

			if onChange != nil {
				onChange(stackName)
			}
		})
	}

	cancelPending := func() {
		debounceMu.Lock()
		for _, t := range pending {
			t.Stop()
		}
		debounceMu.Unlock()
	}

	for {
		select {
		case <-ctx.Done():
			cancelPending()
			return ctx.Err()

		case event, ok := <-watcher.Events:
			if !ok {
				cancelPending()
				return fmt.Errorf("fsnotify events channel closed")
			}

			name := filepath.Base(event.Name)
			dir := filepath.Dir(event.Name)

			// Case 1: Event in the stacks directory itself (new/removed subdirs)
			if dir == stacksDir {
				if event.Op&(fsnotify.Create|fsnotify.Rename) != 0 {
					// Might be a new stack directory — try to watch it
					info, err := os.Stat(event.Name)
					if err == nil && info.IsDir() {
						if err := watcher.Add(event.Name); err != nil {
							slog.Warn("compose watcher: add new subdir", "err", err, "dir", event.Name)
						}
						// Trigger update for the new stack
						triggerUpdate(name)
					}
				}
				if event.Op&fsnotify.Remove != 0 {
					// Stack directory removed — debounce like all other events
					triggerUpdate(name)
				}
				continue
			}

			// Case 2: Event in a stack subdirectory (compose file changed)
			stackName := filepath.Base(dir)
			parentDir := filepath.Dir(dir)

			// Only handle events in direct children of stacksDir
			if parentDir != stacksDir {
				continue
			}

			// Only react to compose file changes
			if !isComposeFile(name) {
				continue
			}

			// Handle write, create, remove, rename events
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
				triggerUpdate(stackName)
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				cancelPending()
				return fmt.Errorf("fsnotify errors channel closed")
			}
			slog.Warn("compose watcher error", "err", err)
		}
	}
}

// isComposeFile checks if a filename matches any accepted compose file name.
func isComposeFile(name string) bool {
	for _, accepted := range acceptedComposeFileNames {
		if name == accepted {
			return true
		}
	}
	return false
}
