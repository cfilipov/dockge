package compose

import (
    "bufio"
    "context"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/cfilipov/dockge/backend-go/internal/docker"
)

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
    m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Starting", stackName, serviceName))
    m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Started", stackName, serviceName))
    m.state.Set(stackName, "running")
    return nil
}

func (m *MockCompose) ServiceStop(_ context.Context, stackName, serviceName string, w io.Writer) error {
    m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Stopping", stackName, serviceName))
    m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Stopped", stackName, serviceName))
    return nil
}

func (m *MockCompose) ServiceRestart(_ context.Context, stackName, serviceName string, w io.Writer) error {
    m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Restarting", stackName, serviceName))
    m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Started", stackName, serviceName))
    return nil
}

func (m *MockCompose) ServicePullAndUp(_ context.Context, stackName, serviceName string, w io.Writer) error {
    m.fakeDelay(w, fmt.Sprintf(" %s Pulling", serviceName))
    m.fakeDelay(w, fmt.Sprintf(" %s Pull complete", serviceName))
    m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Starting", stackName, serviceName))
    m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Started", stackName, serviceName))
    m.state.Set(stackName, "running")
    return nil
}

// --- Internal helpers ---

func (m *MockCompose) up(stackName string, w io.Writer) error {
    services := m.getServices(stackName)
    for _, svc := range services {
        m.fakeDelay(w, fmt.Sprintf(" Network %s_default  Creating", stackName))
        m.fakeDelay(w, fmt.Sprintf(" Network %s_default  Created", stackName))
        m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Creating", stackName, svc))
        m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Created", stackName, svc))
        m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Starting", stackName, svc))
        m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Started", stackName, svc))
    }
    m.state.Set(stackName, "running")
    return nil
}

func (m *MockCompose) stop(stackName, targetService string, w io.Writer) error {
    if targetService != "" {
        m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Stopping", stackName, targetService))
        m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Stopped", stackName, targetService))
    } else {
        for _, svc := range m.getServices(stackName) {
            m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Stopping", stackName, svc))
            m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Stopped", stackName, svc))
        }
        m.state.Set(stackName, "exited")
    }
    return nil
}

func (m *MockCompose) down(stackName string, w io.Writer) error {
    for _, svc := range m.getServices(stackName) {
        m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Stopping", stackName, svc))
        m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Stopped", stackName, svc))
        m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Removing", stackName, svc))
        m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Removed", stackName, svc))
    }
    m.fakeDelay(w, fmt.Sprintf(" Network %s_default  Removing", stackName))
    m.fakeDelay(w, fmt.Sprintf(" Network %s_default  Removed", stackName))
    m.state.Remove(stackName)
    return nil
}

func (m *MockCompose) restart(stackName, targetService string, w io.Writer) error {
    if targetService != "" {
        m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Restarting", stackName, targetService))
        m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Started", stackName, targetService))
    } else {
        for _, svc := range m.getServices(stackName) {
            m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Restarting", stackName, svc))
            m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Started", stackName, svc))
        }
    }
    m.state.Set(stackName, "running")
    return nil
}

func (m *MockCompose) pull(stackName, targetService string, w io.Writer) error {
    services := m.getServices(stackName)
    if targetService != "" {
        services = []string{targetService}
    }
    for _, svc := range services {
        m.fakeDelay(w, fmt.Sprintf(" %s Pulling", svc))
        m.fakeDelay(w, fmt.Sprintf(" %s Pull complete", svc))
    }
    return nil
}

func (m *MockCompose) pause(stackName string, w io.Writer) error {
    for _, svc := range m.getServices(stackName) {
        m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Pausing", stackName, svc))
        m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Paused", stackName, svc))
    }
    return nil
}

func (m *MockCompose) unpause(stackName string, w io.Writer) error {
    for _, svc := range m.getServices(stackName) {
        m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Unpausing", stackName, svc))
        m.fakeDelay(w, fmt.Sprintf(" Container %s-%s-1  Unpaused", stackName, svc))
    }
    m.state.Set(stackName, "running")
    return nil
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

func (m *MockCompose) fakeDelay(w io.Writer, msg string) {
    fmt.Fprintf(w, "%s\r\n", msg)
    time.Sleep(50 * time.Millisecond) // Brief delay for realistic terminal output
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
