package handlers

import (
    "bufio"
    "bytes"
    "context"
    "fmt"
    "log/slog"
    "sort"
    "strings"
    "sync"
    "time"

    "github.com/cfilipov/dockge/internal/terminal"
)

// mainTerminalMu guards mainTerminalName.
var mainTerminalMu sync.Mutex

// ANSI color palette for per-service log prefixes (6 high-contrast colors).
var logColors = [...]string{
    "\033[36m",  // cyan
    "\033[33m",  // yellow
    "\033[32m",  // green
    "\033[35m",  // magenta
    "\033[34m",  // blue
    "\033[91m",  // bright red
}

const colorReset = "\033[0m"

// coloredPrefix returns " serviceName | " with ANSI color, padded to maxLen.
func coloredPrefix(service string, maxLen int, colorIdx int) string {
    color := logColors[colorIdx%len(logColors)]
    return fmt.Sprintf("%s%-*s |%s ", color, maxLen, service, colorReset)
}

// runBanner returns a bold banner line marking a container start boundary.
// Returns empty string if startedAt is zero (mock mode / unknown).
func runBanner(service string, startedAt time.Time) string {
    if startedAt.IsZero() {
        return ""
    }
    // Bold, black text (38;2;0;0;0) on primary-blue background (48;2;116;194;255) — #74c2ff
    return "\n\033[1;38;2;0;0;0;48;2;116;194;255m \u25B6 CONTAINER START \u2014 " + service + " \033[0m\n\n"
}

// stopBanner returns a bold banner line marking a container stop boundary.
func stopBanner(service string) string {
    // Bold, black text (38;2;0;0;0) on warning-yellow background (48;2;248;163;6) — #f8a306
    return "\n\033[1;38;2;0;0;0;48;2;248;163;6m \u25FC CONTAINER STOP \u2014 " + service + " \033[0m\n\n"
}

// runContainerLogLoop streams logs for a single container (by stack+service),
// reconnecting after stop/start cycles. It watches Docker events via the shared
// EventBus to inject start/stop banners and to re-open the log stream when
// the container restarts.
func (app *App) runContainerLogLoop(ctx context.Context, term *terminal.Terminal, termName, stackName, serviceName string) {
    defer app.Terms.RemoveAfter(termName, 30*time.Second)

    containerID, err := app.findContainerID(ctx, stackName, serviceName)
    if err != nil {
        slog.Warn("joinContainerLog: find container", "err", err, "stack", stackName, "service", serviceName)
        term.Write([]byte("[Error] Could not find container for " + serviceName + "\r\n"))
        return
    }

    eventCh, unsub := app.EventBus.Subscribe(64)
    defer unsub()

    lineCh := make(chan []byte, 256)
    go flushLogLines(ctx, term, lineCh)

    tail := "100"
    for {
        // Stream logs until EOF (container stopped or stream closed)
        app.streamContainerLogsToChannel(ctx, containerID, tail, lineCh)

        // After the first stream, only fetch new lines on reconnect
        tail = "0"

        // Wait for the next start event to reconnect
        for {
            select {
            case evt, ok := <-eventCh:
                if !ok {
                    return
                }
                if evt.Project != stackName || evt.Service != serviceName {
                    continue
                }
                switch evt.Action {
                case "die":
                    select {
                    case lineCh <- []byte(stopBanner(serviceName)):
                    case <-ctx.Done():
                        return
                    }
                case "start":
                    startedAt, _ := app.Docker.ContainerStartedAt(ctx, evt.ContainerID)
                    if banner := runBanner(serviceName, startedAt); banner != "" {
                        select {
                        case lineCh <- []byte(banner):
                        case <-ctx.Done():
                            return
                        }
                    }
                    // Re-resolve container ID (may have changed after recreate)
                    containerID = evt.ContainerID
                    goto reconnect
                }
            case <-ctx.Done():
                return
            }
        }
    reconnect:
    }
}

// runContainerLogByNameLoop streams logs for a single container (by name),
// reconnecting after stop/start cycles. Uses the shared EventBus instead of
// opening a dedicated Docker Events connection.
func (app *App) runContainerLogByNameLoop(ctx context.Context, term *terminal.Terminal, termName, containerName string) {
    defer app.Terms.RemoveAfter(termName, 30*time.Second)

    eventCh, unsub := app.EventBus.Subscribe(64)
    defer unsub()

    lineCh := make(chan []byte, 256)
    go flushLogLines(ctx, term, lineCh)

    tail := "100"
    for {
        app.streamContainerLogsToChannel(ctx, containerName, tail, lineCh)
        tail = "0"

        for {
            select {
            case evt, ok := <-eventCh:
                if !ok {
                    return
                }
                evtName := evt.Project + "-" + evt.Service + "-1"
                if evtName != containerName {
                    continue
                }
                switch evt.Action {
                case "die":
                    select {
                    case lineCh <- []byte(stopBanner(containerName)):
                    case <-ctx.Done():
                        return
                    }
                case "start":
                    startedAt, _ := app.Docker.ContainerStartedAt(ctx, containerName)
                    if banner := runBanner(containerName, startedAt); banner != "" {
                        select {
                        case lineCh <- []byte(banner):
                        case <-ctx.Done():
                            return
                        }
                    }
                    goto reconnect
                }
            case <-ctx.Done():
                return
            }
        }
    reconnect:
    }
}

// streamContainerLogsToChannel opens a log stream for a container and sends
// each line to lineCh (for batch flushing) until the stream ends or ctx is cancelled.
func (app *App) streamContainerLogsToChannel(ctx context.Context, containerID, tail string, lineCh chan<- []byte) {
    stream, _, err := app.Docker.ContainerLogs(ctx, containerID, tail, true, false)
    if err != nil {
        if ctx.Err() == nil {
            slog.Warn("container log stream", "err", err, "container", containerID)
            select {
            case lineCh <- []byte("[Error] " + err.Error() + "\r\n"):
            case <-ctx.Done():
            }
        }
        return
    }
    defer stream.Close()

    scanner := bufio.NewScanner(stream)
    scanner.Buffer(make([]byte, 64*1024), 64*1024)
    for scanner.Scan() {
        if ctx.Err() != nil {
            return
        }
        line := make([]byte, len(scanner.Bytes())+1)
        copy(line, scanner.Bytes())
        line[len(line)-1] = '\n'

        select {
        case lineCh <- line:
        case <-ctx.Done():
            return
        }
    }
}

// startCombinedLogs creates a combined log terminal for a stack and starts
// streaming logs using per-container SDK log streams merged through a shared
// channel with a batched flusher. This replaces the previous `docker compose
// logs` subprocess approach, reducing memory from ~30MB to ~200KB per stack.
func (app *App) startCombinedLogs(termName, stackName string) *terminal.Terminal {
    term := app.Terms.Create(termName, terminal.TypePipe)

    ctx, cancel := context.WithCancel(context.Background())
    term.SetCancel(cancel)

    go app.runCombinedLogs(ctx, term, stackName)

    return term
}

// runCombinedLogs orchestrates per-container log readers and a batched flusher.
// It subscribes to Docker events to inject run-boundary banners on restarts and
// spawn new readers when containers are recreated or added. Blocks until ctx is
// cancelled.
func (app *App) runCombinedLogs(ctx context.Context, term *terminal.Terminal, stackName string) {
    containers, err := app.Docker.ContainerList(ctx, true, stackName)
    if err != nil {
        if ctx.Err() == nil {
            slog.Warn("combined logs: list containers", "err", err, "stack", stackName)
            term.Write([]byte("[Error] " + err.Error() + "\r\n"))
        }
        return
    }

    if len(containers) == 0 {
        return
    }

    // Compute max service name length for aligned prefixes
    maxLen := 0
    for _, c := range containers {
        if len(c.Service) > maxLen {
            maxLen = len(c.Service)
        }
    }

    // Assign stable color indices per service
    colorMap := make(map[string]int, len(containers))
    for i, c := range containers {
        colorMap[c.Service] = i
    }

    lineCh := make(chan []byte, 256)

    // Track which container IDs have an active reader goroutine.
    // Docker follow mode continues through restarts (same container ID),
    // so we only spawn new readers for genuinely new container IDs
    // (e.g. after docker compose up --force-recreate).
    var activeReaders sync.Map // containerID → struct{}

    // injectBanner fetches the container's start time and sends a banner.
    injectBanner := func(containerID, service string) {
        startedAt, err := app.Docker.ContainerStartedAt(ctx, containerID)
        if err != nil || startedAt.IsZero() {
            return
        }
        if banner := runBanner(service, startedAt); banner != "" {
            select {
            case lineCh <- []byte(banner):
            case <-ctx.Done():
            }
        }
    }

    // Start the flusher BEFORE writing anything — it drains lineCh and writes
    // to the terminal. Must be a goroutine because the code below writes to
    // lineCh synchronously.
    go flushLogLines(ctx, term, lineCh)

    // Phase 1: Fetch historical logs from all containers with timestamps,
    // merge-sort by timestamp, then write in chronological order. This
    // eliminates the non-deterministic goroutine interleaving that caused
    // flaky E2E screenshots.
    type tsLine struct {
        ts      string // RFC3339Nano prefix (lexicographic sort = chronological)
        display []byte // coloredPrefix + original line + \n
    }
    var allHistorical []tsLine

    for _, c := range containers {
        stream, _, err := app.Docker.ContainerLogs(ctx, c.ID, "100", false, true) // no follow, with timestamps
        if err != nil {
            if ctx.Err() == nil {
                slog.Warn("combined logs: historical fetch", "err", err, "container", c.ID)
            }
            continue
        }
        prefix := coloredPrefix(c.Service, maxLen, colorMap[c.Service])
        scanner := bufio.NewScanner(stream)
        scanner.Buffer(make([]byte, 64*1024), 64*1024)
        for scanner.Scan() {
            raw := scanner.Text()
            // Docker timestamps format: "2024-01-15T10:30:00.123456789Z rest of line"
            ts, line := splitTimestamp(raw)
            display := make([]byte, 0, len(prefix)+len(line)+1)
            display = append(display, prefix...)
            display = append(display, line...)
            display = append(display, '\n')
            allHistorical = append(allHistorical, tsLine{ts: ts, display: display})
        }
        stream.Close()
    }

    // Sort by timestamp (RFC3339Nano strings sort lexicographically)
    sort.Slice(allHistorical, func(i, j int) bool {
        return allHistorical[i].ts < allHistorical[j].ts
    })

    // Write sorted historical lines
    for _, l := range allHistorical {
        select {
        case lineCh <- l.display:
        case <-ctx.Done():
            return
        }
    }

    // Phase 2: Spawn parallel follow goroutines (tail=0, no timestamps).
    // Lines arrive in real-time so ordering is naturally correct.
    for _, c := range containers {
        wasRunning := c.State == "running"
        activeReaders.Store(c.ID, struct{}{})
        go func(id, svc string, idx int, running bool) {
            defer activeReaders.Delete(id)
            app.readContainerLogs(ctx, id, svc, maxLen, idx, "0", true, running, lineCh)
        }(c.ID, c.Service, colorMap[c.Service], wasRunning)
    }

    // Watch for container events via the shared EventBus:
    // - "start": inject banner + spawn reader for new container IDs
    // - "die": stop banner is injected by readContainerLogs after stream EOF
    eventCh, unsub := app.EventBus.Subscribe(64)
    defer unsub()

    for {
        select {
        case evt, ok := <-eventCh:
            if !ok {
                return
            }
            if evt.Project != stackName {
                continue
            }
            switch evt.Action {
            case "start":
                idx, known := colorMap[evt.Service]
                if !known {
                    idx = len(colorMap)
                    colorMap[evt.Service] = idx
                    if len(evt.Service) > maxLen {
                        maxLen = len(evt.Service)
                    }
                }
                injectBanner(evt.ContainerID, evt.Service)
                // Spawn reader only for new container IDs (recreated containers)
                if _, loaded := activeReaders.LoadOrStore(evt.ContainerID, struct{}{}); !loaded {
                    go func(id, svc string, ci int) {
                        defer activeReaders.Delete(id)
                        app.readContainerLogs(ctx, id, svc, maxLen, ci, "0", true, true, lineCh)
                    }(evt.ContainerID, evt.Service, idx)
                }
            }
        case <-ctx.Done():
            return
        }
    }
}

// readContainerLogs streams logs for a single container, prefixing each line
// with a colored service name. Runs until the stream closes or ctx is cancelled.
// Start banners are injected by the caller. Stop banners are injected here
// after the stream ends (ensuring they appear after all shutdown log output).
// wasRunning indicates the container was running when the reader started — if
// true, a stop banner is injected after EOF (the container transitioned to
// stopped). If false (container was already stopped), no banner is shown.
// Use tail="100" for initial readers (show history) and tail="0" for
// event-spawned readers (follow only).
func (app *App) readContainerLogs(ctx context.Context, containerID, service string, maxLen, colorIdx int, tail string, follow, wasRunning bool, lineCh chan<- []byte) {
    stream, _, err := app.Docker.ContainerLogs(ctx, containerID, tail, follow, false)
    if err != nil {
        if ctx.Err() == nil {
            slog.Warn("combined logs: container stream", "err", err, "container", containerID)
        }
        return
    }
    defer stream.Close()

    prefix := coloredPrefix(service, maxLen, colorIdx)

    scanner := bufio.NewScanner(stream)
    scanner.Buffer(make([]byte, 64*1024), 64*1024)
    for scanner.Scan() {
        line := make([]byte, 0, len(prefix)+len(scanner.Bytes())+1)
        line = append(line, prefix...)
        line = append(line, scanner.Bytes()...)
        line = append(line, '\n')

        select {
        case lineCh <- line:
        case <-ctx.Done():
            return
        }
    }
    // Inject stop banner after EOF if the container was running when we
    // started. This guarantees the banner follows all shutdown log output
    // (both are sequential on this goroutine's channel sends).
    if wasRunning && ctx.Err() == nil {
        select {
        case lineCh <- []byte(stopBanner(service)):
        case <-ctx.Done():
        }
    }
}

// flushLogLines drains lineCh in batches on a 50ms tick and writes them to the
// terminal. This coalesces many small per-line writes into fewer, larger writes
// which reduces WebSocket message volume.
func flushLogLines(ctx context.Context, term *terminal.Terminal, lineCh <-chan []byte) {
    ticker := time.NewTicker(50 * time.Millisecond)
    defer ticker.Stop()

    var batch bytes.Buffer
    batch.Grow(4096)

    drain := func() bool {
        for {
            select {
            case line, ok := <-lineCh:
                if !ok {
                    return false
                }
                batch.Write(line)
            default:
                return true
            }
        }
    }

    flush := func() {
        if batch.Len() > 0 {
            term.Write(batch.Bytes())
            batch.Reset()
        }
    }

    for {
        select {
        case <-ctx.Done():
            drain()
            flush()
            return
        case <-ticker.C:
            if !drain() {
                flush()
                return
            }
            flush()
        }
    }
}

// findContainerID resolves a stack+service name to a container ID by querying
// the Docker client.
func (app *App) findContainerID(ctx context.Context, stackName, serviceName string) (string, error) {
    containers, err := app.Docker.ContainerList(ctx, true, stackName)
    if err != nil {
        return "", err
    }
    for _, c := range containers {
        if c.Service == serviceName {
            return c.ID, nil
        }
    }
    // Fallback: use container name (for mock mode where labels may not be set)
    for _, c := range containers {
        if strings.Contains(c.Name, serviceName) {
            return c.ID, nil
        }
    }
    // Last resort: return the name as-is (docker CLI will resolve it)
    return stackName + "-" + serviceName + "-1", nil
}

// splitTimestamp splits a Docker log line with timestamps enabled into the
// timestamp prefix and the remaining content. Docker prefixes each line with
// an RFC3339Nano timestamp followed by a space, e.g.:
// "2024-01-15T10:30:00.123456789Z actual log line content"
// If no timestamp is found, returns ("", raw) so sorting still works (empty
// strings sort first).
func splitTimestamp(raw string) (ts, line string) {
    if idx := strings.IndexByte(raw, ' '); idx > 0 && idx <= 35 {
        // Sanity check: RFC3339Nano timestamps are 20-35 chars
        return raw[:idx], raw[idx+1:]
    }
    return "", raw
}

// extractCombinedStackName extracts the stack name from a combined terminal name.
// Format: "combined-{stackName}"
func extractCombinedStackName(termName string) string {
    return strings.TrimPrefix(termName, "combined-")
}

