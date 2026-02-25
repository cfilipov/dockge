package compose

import (
    "bufio"
    "bytes"
    "context"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/cfilipov/dockge/internal/docker"
)

// ANSI escape sequences for Docker Compose v2 style output.
const (
    ansiGreen   = "\033[32m"
    ansiReset   = "\033[0m"
    ansiHideCur = "\033[?25l"
    ansiShowCur = "\033[?25h"
    ansiCurUp   = "\033[A"
    ansiEraseLn = "\033[2K"
)

// spinnerFrames matches the Braille spinner used by Docker Compose v2.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// progressTask represents a single line in the Docker Compose progress display.
type progressTask struct {
    name   string // e.g. "Container web-app-nginx-1" or "Network web-app_default"
    action string // in-progress verb: "Creating", "Starting", etc.
    done   string // completed verb: "Created", "Started", etc.
}

// progressRenderer draws Docker Compose v2 style animated progress output.
type progressRenderer struct {
    w     io.Writer
    verb  string // header verb: "Running", "Restarting", etc.
    tasks []progressTask
    delay time.Duration // per-frame delay
}

// render draws the full animated progress sequence to w.
// It shows a header line and one line per task. Tasks complete one at a time,
// with spinner animation on pending/in-progress tasks and green checkmarks on
// completed ones.
func (r *progressRenderer) render() {
    n := len(r.tasks)
    if n == 0 {
        return
    }

    // Number of spinner frames to show per task before marking it done.
    const framesPerTask = 3

    // Track when each task starts spinning for elapsed time.
    taskStart := make([]time.Time, n)
    taskElapsed := make([]time.Duration, n)

    // Hide cursor.
    fmt.Fprint(r.w, ansiHideCur)

    // Draw initial frame: all tasks pending.
    r.writeHeader(0)
    spinIdx := 0
    for i := range r.tasks {
        r.writeTaskPending(i, spinIdx)
    }

    // Animate: complete tasks one at a time.
    for completed := 0; completed < n; completed++ {
        taskStart[completed] = time.Now()

        // Show spinner frames for this task.
        for frame := 0; frame < framesPerTask; frame++ {
            time.Sleep(r.delay)
            spinIdx++
            r.moveCursorUp(n + 1) // +1 for header
            r.writeHeader(completed)
            for i := range r.tasks {
                if i < completed {
                    r.writeTaskDone(i, taskElapsed[i])
                } else {
                    r.writeTaskPending(i, spinIdx)
                }
            }
        }

        // Mark this task as done.
        taskElapsed[completed] = time.Since(taskStart[completed])
        time.Sleep(r.delay)
        r.moveCursorUp(n + 1)
        r.writeHeader(completed + 1)
        for i := range r.tasks {
            if i <= completed {
                r.writeTaskDone(i, taskElapsed[i])
            } else {
                r.writeTaskPending(i, spinIdx)
            }
        }
    }

    // Show cursor.
    fmt.Fprint(r.w, ansiShowCur)
}

func (r *progressRenderer) writeHeader(completed int) {
    total := len(r.tasks)
    fmt.Fprintf(r.w, "\r%s %s[+]%s %s %d/%d\r\n",
        ansiEraseLn, ansiGreen, ansiReset, r.verb, completed, total)
}

func (r *progressRenderer) writeTaskPending(i, spinIdx int) {
    frame := spinnerFrames[spinIdx%len(spinnerFrames)]
    fmt.Fprintf(r.w, "\r%s %s %s  %s\r\n",
        ansiEraseLn, frame, r.tasks[i].name, r.tasks[i].action)
}

func (r *progressRenderer) writeTaskDone(i int, elapsed time.Duration) {
    secs := elapsed.Seconds()
    fmt.Fprintf(r.w, "\r%s %s✔%s %s  %-12s %.1fs\r\n",
        ansiEraseLn, ansiGreen, ansiReset,
        r.tasks[i].name, r.tasks[i].done, secs)
}

func (r *progressRenderer) moveCursorUp(lines int) {
    for i := 0; i < lines; i++ {
        fmt.Fprint(r.w, ansiCurUp)
    }
}

// MockCompose implements Composer as a pure in-memory mock.
// Stack state is tracked via a shared MockState (in-memory map)
// so it stays in sync with MockClient.
type MockCompose struct {
    StacksDir string
    state     *docker.MockState
}

// NewMockCompose creates a new mock compose executor.
func NewMockCompose(stacksDir string, state *docker.MockState) *MockCompose {
    return &MockCompose{
        StacksDir: stacksDir,
        state:     state,
    }
}

// Ensure MockCompose implements Composer at compile time.
var _ Composer = (*MockCompose)(nil)

func (m *MockCompose) RunCompose(ctx context.Context, stackName string, w io.Writer, args ...string) error {
    if len(args) == 0 {
        return fmt.Errorf("no compose command specified")
    }

    subcmd := args[0]
    switch subcmd {
    case "up":
        return m.up(stackName, w)
    case "stop":
        svc := findServiceArg(args[1:])
        return m.stop(stackName, svc, w)
    case "down":
        return m.down(stackName, w)
    case "restart":
        svc := findServiceArg(args[1:])
        return m.restart(stackName, svc, w)
    case "pull":
        svc := findServiceArg(args[1:])
        return m.pull(stackName, svc, w)
    case "pause":
        return m.pause(stackName, w)
    case "unpause":
        return m.unpause(stackName, w)
    case "logs":
        return m.logs(ctx, stackName, w, args[1:])
    case "config":
        return m.config(stackName, w)
    default:
        fmt.Fprintf(w, "[mock] unsupported compose command: %s\r\n", subcmd)
        return nil
    }
}

func (m *MockCompose) RunDocker(ctx context.Context, stackName string, w io.Writer, args ...string) error {
    if len(args) >= 2 && args[0] == "image" && args[1] == "prune" {
        fmt.Fprintf(w, "Total reclaimed space: 0B\r\n")
        return nil
    }
    fmt.Fprintf(w, "[mock] docker %s\r\n", strings.Join(args, " "))
    return nil
}

func (m *MockCompose) Config(_ context.Context, stackName string, w io.Writer) error {
    return m.config(stackName, w)
}

func (m *MockCompose) DownRemoveOrphans(_ context.Context, stackName string, w io.Writer) error {
    return m.down(stackName, w)
}

func (m *MockCompose) DownVolumes(_ context.Context, stackName string, w io.Writer) error {
    return m.down(stackName, w)
}

func (m *MockCompose) ServiceUp(_ context.Context, stackName, serviceName string, w io.Writer) error {
    r := &progressRenderer{
        w:     w,
        verb:  "Running",
        delay: 50 * time.Millisecond,
        tasks: []progressTask{
            {name: fmt.Sprintf("Container %s-%s-1", stackName, serviceName), action: "Starting", done: "Started"},
        },
    }
    r.render()
    m.state.Set(stackName, "running")
    return nil
}

func (m *MockCompose) ServiceStop(_ context.Context, stackName, serviceName string, w io.Writer) error {
    r := &progressRenderer{
        w:     w,
        verb:  "Stopping",
        delay: 50 * time.Millisecond,
        tasks: []progressTask{
            {name: fmt.Sprintf("Container %s-%s-1", stackName, serviceName), action: "Stopping", done: "Stopped"},
        },
    }
    r.render()
    return nil
}

func (m *MockCompose) ServiceRestart(_ context.Context, stackName, serviceName string, w io.Writer) error {
    r := &progressRenderer{
        w:     w,
        verb:  "Restarting",
        delay: 50 * time.Millisecond,
        tasks: []progressTask{
            {name: fmt.Sprintf("Container %s-%s-1", stackName, serviceName), action: "Restarting", done: "Started"},
        },
    }
    r.render()
    return nil
}

func (m *MockCompose) ServicePullAndUp(_ context.Context, stackName, serviceName string, w io.Writer) error {
    r := &progressRenderer{
        w:     w,
        verb:  "Pulling",
        delay: 50 * time.Millisecond,
        tasks: []progressTask{
            {name: serviceName, action: "Pulling", done: "Pulled"},
            {name: fmt.Sprintf("Container %s-%s-1", stackName, serviceName), action: "Starting", done: "Started"},
        },
    }
    r.render()
    m.state.Set(stackName, "running")
    return nil
}

// --- Internal helpers ---

func (m *MockCompose) up(stackName string, w io.Writer) error {
    services := m.getServices(stackName)

    var tasks []progressTask
    // Network creation first.
    tasks = append(tasks, progressTask{
        name:   fmt.Sprintf("Network %s_default", stackName),
        action: "Creating",
        done:   "Created",
    })
    // Then per-service: create + start.
    for _, svc := range services {
        tasks = append(tasks,
            progressTask{
                name:   fmt.Sprintf("Container %s-%s-1", stackName, svc),
                action: "Creating",
                done:   "Created",
            },
            progressTask{
                name:   fmt.Sprintf("Container %s-%s-1", stackName, svc),
                action: "Starting",
                done:   "Started",
            },
        )
    }

    r := &progressRenderer{w: w, verb: "Running", delay: 50 * time.Millisecond, tasks: tasks}
    r.render()
    m.state.Set(stackName, "running")
    return nil
}

func (m *MockCompose) stop(stackName, targetService string, w io.Writer) error {
    var tasks []progressTask
    if targetService != "" {
        tasks = []progressTask{
            {name: fmt.Sprintf("Container %s-%s-1", stackName, targetService), action: "Stopping", done: "Stopped"},
        }
    } else {
        for _, svc := range m.getServices(stackName) {
            tasks = append(tasks, progressTask{
                name:   fmt.Sprintf("Container %s-%s-1", stackName, svc),
                action: "Stopping",
                done:   "Stopped",
            })
        }
    }

    r := &progressRenderer{w: w, verb: "Stopping", delay: 50 * time.Millisecond, tasks: tasks}
    r.render()
    if targetService == "" {
        m.state.Set(stackName, "exited")
    }
    return nil
}

func (m *MockCompose) down(stackName string, w io.Writer) error {
    var tasks []progressTask
    // Per-service: stop + remove.
    for _, svc := range m.getServices(stackName) {
        tasks = append(tasks,
            progressTask{
                name:   fmt.Sprintf("Container %s-%s-1", stackName, svc),
                action: "Stopping",
                done:   "Stopped",
            },
            progressTask{
                name:   fmt.Sprintf("Container %s-%s-1", stackName, svc),
                action: "Removing",
                done:   "Removed",
            },
        )
    }
    // Network removal last.
    tasks = append(tasks, progressTask{
        name:   fmt.Sprintf("Network %s_default", stackName),
        action: "Removing",
        done:   "Removed",
    })

    r := &progressRenderer{w: w, verb: "Running", delay: 50 * time.Millisecond, tasks: tasks}
    r.render()
    m.state.Remove(stackName)
    return nil
}

func (m *MockCompose) restart(stackName, targetService string, w io.Writer) error {
    var tasks []progressTask
    if targetService != "" {
        tasks = []progressTask{
            {name: fmt.Sprintf("Container %s-%s-1", stackName, targetService), action: "Restarting", done: "Started"},
        }
    } else {
        for _, svc := range m.getServices(stackName) {
            tasks = append(tasks, progressTask{
                name:   fmt.Sprintf("Container %s-%s-1", stackName, svc),
                action: "Restarting",
                done:   "Started",
            })
        }
    }

    r := &progressRenderer{w: w, verb: "Restarting", delay: 50 * time.Millisecond, tasks: tasks}
    r.render()
    m.state.Set(stackName, "running")
    return nil
}

func (m *MockCompose) pull(stackName, targetService string, w io.Writer) error {
    services := m.getServices(stackName)
    if targetService != "" {
        services = []string{targetService}
    }

    var tasks []progressTask
    for _, svc := range services {
        tasks = append(tasks, progressTask{
            name:   svc,
            action: "Pulling",
            done:   "Pulled",
        })
    }

    r := &progressRenderer{w: w, verb: "Pulling", delay: 50 * time.Millisecond, tasks: tasks}
    r.render()
    return nil
}

func (m *MockCompose) pause(stackName string, w io.Writer) error {
    var tasks []progressTask
    for _, svc := range m.getServices(stackName) {
        tasks = append(tasks, progressTask{
            name:   fmt.Sprintf("Container %s-%s-1", stackName, svc),
            action: "Pausing",
            done:   "Paused",
        })
    }

    r := &progressRenderer{w: w, verb: "Pausing", delay: 50 * time.Millisecond, tasks: tasks}
    r.render()
    return nil
}

func (m *MockCompose) unpause(stackName string, w io.Writer) error {
    var tasks []progressTask
    for _, svc := range m.getServices(stackName) {
        tasks = append(tasks, progressTask{
            name:   fmt.Sprintf("Container %s-%s-1", stackName, svc),
            action: "Unpausing",
            done:   "Unpaused",
        })
    }

    r := &progressRenderer{w: w, verb: "Unpausing", delay: 50 * time.Millisecond, tasks: tasks}
    r.render()
    m.state.Set(stackName, "running")
    return nil
}

// logColors mirrors docker compose's service name color palette.
var logColors = []string{
    "\033[36m", // cyan
    "\033[33m", // yellow
    "\033[32m", // green
    "\033[35m", // magenta
    "\033[34m", // blue
    "\033[96m", // bright cyan
    "\033[93m", // bright yellow
    "\033[92m", // bright green
    "\033[95m", // bright magenta
    "\033[94m", // bright blue
}

func (m *MockCompose) logs(ctx context.Context, stackName string, w io.Writer, args []string) error {
    services := m.getServices(stackName)
    if len(services) == 0 {
        return nil
    }

    // Compute max service name length for aligned prefixes.
    maxLen := 0
    for _, svc := range services {
        if len(svc) > maxLen {
            maxLen = len(svc)
        }
    }

    // Build all initial log lines in a single buffer so they arrive as one
    // write to the terminal, avoiding N individual WebSocket messages.
    var buf bytes.Buffer
    for i, svc := range services {
        color := logColors[i%len(logColors)]
        padded := fmt.Sprintf("%-*s", maxLen, svc)
        prefix := color + padded + " | " + "\033[0m"
        for line := 1; line <= 3; line++ {
            fmt.Fprintf(&buf, "%s[mock] log line %d from %s\n", prefix, line, svc)
        }
    }
    w.Write(buf.Bytes())

    // If -f/--follow, block until context is cancelled (simulates tailing).
    if hasFlag(args, "-f") || hasFlag(args, "--follow") {
        <-ctx.Done()
    }
    return nil
}

// hasFlag checks whether a flag is present in args.
func hasFlag(args []string, flag string) bool {
    for _, a := range args {
        if a == flag {
            return true
        }
    }
    return false
}

func (m *MockCompose) config(stackName string, w io.Writer) error {
    composeFile := m.findComposeFile(stackName)
    if composeFile == "" {
        return fmt.Errorf("no configuration file provided: not found")
    }

    // Check that it has a "services:" key
    f, err := os.Open(composeFile)
    if err != nil {
        return fmt.Errorf("no configuration file provided: not found")
    }
    defer f.Close()

    scanner := bufio.NewScanner(f)
    hasServices := false
    for scanner.Scan() {
        if strings.TrimSpace(scanner.Text()) == "services:" {
            hasServices = true
            break
        }
    }
    if !hasServices {
        return fmt.Errorf("services must be a mapping")
    }
    return nil
}

func (m *MockCompose) findComposeFile(stackName string) string {
    for _, name := range []string{"compose.yaml", "docker-compose.yaml", "docker-compose.yml", "compose.yml"} {
        path := filepath.Join(m.StacksDir, stackName, name)
        if _, err := os.Stat(path); err == nil {
            return path
        }
    }
    return ""
}

func (m *MockCompose) getServices(stackName string) []string {
    composeFile := m.findComposeFile(stackName)
    if composeFile == "" {
        return nil
    }

    f, err := os.Open(composeFile)
    if err != nil {
        return nil
    }
    defer f.Close()

    var services []string
    inServices := false
    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        line := scanner.Text()
        trimmed := strings.TrimRight(line, " \t")
        if trimmed == "services:" {
            inServices = true
            continue
        }
        if !inServices {
            continue
        }
        if len(trimmed) > 0 && trimmed[0] != ' ' && trimmed[0] != '#' {
            break
        }
        if len(line) > 2 && line[0] == ' ' && line[1] == ' ' && line[2] != ' ' && strings.HasSuffix(trimmed, ":") {
            svc := strings.TrimSpace(strings.TrimSuffix(trimmed, ":"))
            services = append(services, svc)
        }
    }
    return services
}

// findServiceArg extracts a service name from compose args, skipping flags.
func findServiceArg(args []string) string {
    for _, a := range args {
        if !strings.HasPrefix(a, "-") {
            return a
        }
    }
    return ""
}
