package docker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MockClient implements Client as a pure in-memory mock for development
// environments without a real Docker daemon. It synthesizes container data
// by scanning compose.yaml files on disk and tracking state via a shared
// MockState (in-memory map).
type MockClient struct {
	stacksDir string
	state     *MockState
}

// NewMockClient returns a new in-memory MockClient.
func NewMockClient(stacksDir string, state *MockState) *MockClient {
	return &MockClient{
		stacksDir: stacksDir,
		state:     state,
	}
}

func (m *MockClient) ContainerList(ctx context.Context, all bool, projectFilter string) ([]Container, error) {
	entries, err := os.ReadDir(m.stacksDir)
	if err != nil {
		return nil, fmt.Errorf("scan stacks dir: %w", err)
	}

	var containers []Container
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		stackName := entry.Name()

		// Apply project filter
		if projectFilter != "" && stackName != projectFilter {
			continue
		}

		// Check compose file exists
		composeFile := m.findComposeFile(stackName)
		if composeFile == "" {
			continue
		}

		// Check stack state
		status := m.state.Get(stackName)
		if status == "inactive" {
			continue
		}

		services := m.getServices(stackName, composeFile)
		for _, svc := range services {
			svcState := m.getServiceState(stackName, svc, status)

			// Skip stopped containers unless all=true
			if !all && svcState != "running" {
				continue
			}

			image := m.getRunningImage(stackName, svc, composeFile)
			containers = append(containers, Container{
				ID:      fmt.Sprintf("mock-%s-%s-1", stackName, svc),
				Name:    fmt.Sprintf("%s-%s-1", stackName, svc),
				Project: stackName,
				Service: svc,
				Image:   image,
				State:   svcState,
				Health:  m.getServiceHealth(stackName, svc),
			})
		}
	}
	return containers, nil
}

func (m *MockClient) ContainerInspect(_ context.Context, id string) (string, error) {
	// Return a minimal mock inspect response
	cleanID := strings.TrimPrefix(id, "mock-")
	return fmt.Sprintf(`[{
    "Id": "%s",
    "Created": "2026-02-18T00:00:00.000000000Z",
    "Name": "/%s",
    "State": {
        "Status": "running",
        "Running": true,
        "Paused": false,
        "Restarting": false,
        "OOMKilled": false,
        "Dead": false,
        "Pid": 12345,
        "ExitCode": 0,
        "StartedAt": "2026-02-18T00:00:00.000000000Z"
    },
    "Image": "sha256:mock-image-hash",
    "Config": {
        "Hostname": "%s",
        "Image": "mock-image:latest",
        "Env": ["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"]
    },
    "NetworkSettings": {
        "Networks": {
            "bridge": {
                "IPAddress": "172.17.0.2",
                "Gateway": "172.17.0.1"
            }
        }
    }
}]`, id, cleanID, cleanID), nil
}

func (m *MockClient) ContainerStats(_ context.Context, projectFilter string) (map[string]ContainerStat, error) {
	entries, err := os.ReadDir(m.stacksDir)
	if err != nil {
		return nil, err
	}

	result := make(map[string]ContainerStat)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		stackName := entry.Name()
		if projectFilter != "" && stackName != projectFilter {
			continue
		}
		status := m.state.Get(stackName)
		if status != "running" {
			continue
		}

		composeFile := m.findComposeFile(stackName)
		if composeFile == "" {
			continue
		}
		services := m.getServices(stackName, composeFile)
		for _, svc := range services {
			name := fmt.Sprintf("%s-%s-1", stackName, svc)
			result[name] = ContainerStat{
				Name:     name,
				CPUPerc:  "0.12%",
				MemPerc:  "1.25%",
				MemUsage: "24MiB / 2GiB",
				NetIO:    "1.5kB / 900B",
				BlockIO:  "0B / 0B",
				PIDs:     "5",
			}
		}
	}
	return result, nil
}

func (m *MockClient) ContainerLogs(_ context.Context, containerID string, tail string, follow bool) (io.ReadCloser, bool, error) {
	cleanID := strings.TrimPrefix(containerID, "mock-")
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		fmt.Fprintf(pw, "[mock] Log output for container %s\n", cleanID)
		fmt.Fprintf(pw, "[mock] Container started successfully\n")

		if follow {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				_, err := fmt.Fprintf(pw, "[mock] %s heartbeat for %s\n", time.Now().Format(time.RFC3339), cleanID)
				if err != nil {
					return // pipe closed
				}
			}
		}
	}()

	return pr, false, nil
}

func (m *MockClient) ImageInspect(_ context.Context, imageRef string) ([]string, error) {
	repo, tag := splitImageRef(imageRef)
	hash := mockHash(repo + ":" + tag)
	digest := fmt.Sprintf("sha256:%s%s", hash, hash)
	return []string{fmt.Sprintf("%s@%s", repo, digest)}, nil
}

func (m *MockClient) DistributionInspect(_ context.Context, imageRef string) (string, error) {
	repo, tag := splitImageRef(imageRef)

	// For images with "updates" (nginx, wordpress, postgres), return a different digest
	var hash string
	switch repo {
	case "nginx", "wordpress", "postgres":
		hash = mockHash(repo + ":" + tag + ":remote-newer")
	default:
		hash = mockHash(repo + ":" + tag)
	}
	return fmt.Sprintf("sha256:%s%s", hash, hash), nil
}

func (m *MockClient) ImagePrune(_ context.Context, all bool) (string, error) {
	return "Total reclaimed space: 0B", nil
}

func (m *MockClient) NetworkList(_ context.Context) ([]string, error) {
	return []string{"bridge", "host", "none"}, nil
}

// Events synthesizes container events by polling the in-memory state every 60s
// and diffing against previous state.
func (m *MockClient) Events(ctx context.Context) (<-chan ContainerEvent, <-chan error) {
	events := make(chan ContainerEvent, 32)
	errs := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errs)

		prev := m.snapshot()

		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				curr := m.snapshot()
				// Find new or changed containers
				for key, c := range curr {
					old, existed := prev[key]
					if !existed {
						select {
						case events <- ContainerEvent{Action: "start", Project: c.Project, Service: c.Service, ContainerID: c.ID}:
						case <-ctx.Done():
							return
						}
					} else if old.State != c.State {
						action := "die"
						if c.State == "running" {
							action = "start"
						} else if c.State == "paused" {
							action = "pause"
						}
						select {
						case events <- ContainerEvent{Action: action, Project: c.Project, Service: c.Service, ContainerID: c.ID}:
						case <-ctx.Done():
							return
						}
					}
				}
				// Find removed containers
				for key, c := range prev {
					if _, exists := curr[key]; !exists {
						select {
						case events <- ContainerEvent{Action: "destroy", Project: c.Project, Service: c.Service, ContainerID: c.ID}:
						case <-ctx.Done():
							return
						}
					}
				}
				prev = curr
			}
		}
	}()

	return events, errs
}

func (m *MockClient) Close() error {
	return nil
}

// --- Internal helpers ---

func (m *MockClient) findComposeFile(stackName string) string {
	for _, name := range []string{"compose.yaml", "docker-compose.yaml", "docker-compose.yml", "compose.yml"} {
		path := filepath.Join(m.stacksDir, stackName, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func (m *MockClient) getServices(stackName, composeFile string) []string {
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

func (m *MockClient) getServiceState(stackName, svc, stackStatus string) string {
	if stackStatus != "running" {
		return "exited"
	}
	// Hardcoded mock behaviors for featured stacks
	if stackName == "01-web-app" && svc == "redis" {
		return "exited"
	}
	if stackName == "06-mixed-state" && svc == "worker" {
		return "exited"
	}
	// Some filler stacks with multiple services get a mix of running/exited
	// (indices ending in 8 or 9 have >1 service; make the second service exited
	// for indices divisible by 20)
	if strings.HasPrefix(stackName, "stack-") {
		var idx int
		if _, err := fmt.Sscanf(stackName, "stack-%d", &idx); err == nil {
			if idx%20 == 8 && svc != "" && strings.HasSuffix(svc, "-1") {
				return "exited"
			}
		}
	}
	return "running"
}

func (m *MockClient) getServiceHealth(stackName, svc string) string {
	// Simulate unhealthy services for specific stacks
	if stackName == "05-multi-service" && svc == "db" {
		return "unhealthy"
	}
	// Some filler stacks get unhealthy status (indices ending in 7, mod 30 == 17)
	if strings.HasPrefix(stackName, "stack-") {
		var idx int
		if _, err := fmt.Sscanf(stackName, "stack-%d", &idx); err == nil {
			if idx%30 == 17 && svc != "" {
				return "unhealthy"
			}
		}
	}
	return ""
}

func (m *MockClient) getRunningImage(stackName, svc, composeFile string) string {
	// Simulate recreateNecessary for specific stacks
	if stackName == "02-blog" && svc == "wordpress" {
		return "wordpress:6.3"
	}
	if stackName == "01-web-app" && svc == "nginx" {
		return "nginx:1.24"
	}
	// Default: parse compose.yaml for the image
	return m.getComposeImage(svc, composeFile)
}

func (m *MockClient) getComposeImage(svc, composeFile string) string {
	f, err := os.Open(composeFile)
	if err != nil {
		return "mock-image:latest"
	}
	defer f.Close()

	inTarget := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimRight(line, " \t")
		// Match service declaration
		if len(line) > 2 && line[0] == ' ' && line[1] == ' ' && line[2] != ' ' && strings.HasSuffix(trimmed, ":") {
			name := strings.TrimSpace(strings.TrimSuffix(trimmed, ":"))
			inTarget = (name == svc)
			continue
		}
		if inTarget && strings.Contains(line, "image:") {
			parts := strings.SplitN(line, "image:", 2)
			if len(parts) == 2 {
				img := strings.TrimSpace(parts[1])
				if img != "" {
					return img
				}
			}
		}
	}
	return "mock-image:latest"
}

func (m *MockClient) snapshot() map[string]Container {
	containers, _ := m.ContainerList(context.Background(), true, "")
	result := make(map[string]Container, len(containers))
	for _, c := range containers {
		result[c.ID] = c
	}
	return result
}

// splitImageRef splits "repo:tag" into repo and tag. Defaults tag to "latest".
func splitImageRef(ref string) (string, string) {
	// Handle digest references (repo@sha256:...)
	if idx := strings.Index(ref, "@"); idx >= 0 {
		return ref[:idx], "latest"
	}
	if idx := strings.LastIndex(ref, ":"); idx >= 0 {
		return ref[:idx], ref[idx+1:]
	}
	return ref, "latest"
}

// mockHash generates a deterministic 32-char hex hash from a string.
// Uses a simple FNV-like approach for reproducibility.
func mockHash(s string) string {
	var h uint64 = 14695981039346656037 // FNV offset basis
	for _, c := range s {
		h ^= uint64(c)
		h *= 1099511628211 // FNV prime
	}
	return fmt.Sprintf("%016x%016x", h, h^0xdeadbeefcafebabe)
}

// Ensure MockClient implements Client at compile time.
var _ Client = (*MockClient)(nil)
