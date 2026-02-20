package compose

import (
    "context"
    "log/slog"
    "os"
    "path/filepath"
    "sync"
    "time"

    "github.com/fsnotify/fsnotify"
)

// StartWatcher watches the stacks directory tree for compose file changes.
// On change, re-parses the affected stack's compose file and updates the cache.
// Calls onChange(stackName) after each update so the caller can broadcast.
func StartWatcher(ctx context.Context, stacksDir string, cache *ComposeCache, onChange func(stackName string)) error {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return err
    }

    // Watch the top-level stacks directory (for new/removed stack subdirs)
    if err := watcher.Add(stacksDir); err != nil {
        watcher.Close()
        return err
    }

    // Watch each existing stack subdirectory
    entries, err := os.ReadDir(stacksDir)
    if err != nil {
        watcher.Close()
        return err
    }
    for _, entry := range entries {
        if entry.IsDir() {
            subdir := filepath.Join(stacksDir, entry.Name())
            if err := watcher.Add(subdir); err != nil {
                slog.Warn("compose watcher: add subdir", "err", err, "dir", subdir)
            }
        }
    }

    go runWatcher(ctx, watcher, stacksDir, cache, onChange)

    slog.Info("compose file watcher started", "dir", stacksDir)
    return nil
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

// runWatcher is the main loop for the fsnotify watcher.
func runWatcher(ctx context.Context, watcher *fsnotify.Watcher, stacksDir string, cache *ComposeCache, onChange func(stackName string)) {
    defer watcher.Close()

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

            path := FindComposeFile(stacksDir, stackName)
            if path == "" {
                // Compose file removed — delete from cache
                cache.Delete(stackName)
                slog.Debug("compose watcher: file removed", "stack", stackName)
            } else {
                services := ParseFile(path)
                if services != nil {
                    cache.Update(stackName, services)
                    slog.Debug("compose watcher: file updated", "stack", stackName, "services", len(services))
                }
            }

            if onChange != nil {
                onChange(stackName)
            }
        })
    }

    for {
        select {
        case <-ctx.Done():
            // Cancel all pending timers
            debounceMu.Lock()
            for _, t := range pending {
                t.Stop()
            }
            debounceMu.Unlock()
            return

        case event, ok := <-watcher.Events:
            if !ok {
                return
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
                        // Parse if compose file exists
                        triggerUpdate(name)
                    }
                }
                if event.Op&fsnotify.Remove != 0 {
                    // Stack directory removed
                    cache.Delete(name)
                    if onChange != nil {
                        onChange(name)
                    }
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
                return
            }
            slog.Warn("compose watcher error", "err", err)
        }
    }
}

