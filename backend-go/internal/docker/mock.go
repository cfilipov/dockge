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

	// Append standalone containers (no compose project) when not filtering by project
	if projectFilter == "" {
		standalones := []struct {
			name  string
			image string
			state string
		}{
			{"portainer", "portainer/portainer-ce:latest", "running"},
			{"watchtower", "containrrr/watchtower:latest", "running"},
			{"homeassistant", "ghcr.io/home-assistant/home-assistant:stable", "exited"},
		}
		for _, s := range standalones {
			if !all && s.state != "running" {
				continue
			}
			containers = append(containers, Container{
				ID:      fmt.Sprintf("mock-standalone-%s", s.name),
				Name:    s.name,
				Project: "",
				Service: "",
				Image:   s.image,
				State:   s.state,
				Health:  "",
			})
		}
	}

	return containers, nil
}

func (m *MockClient) ContainerInspect(_ context.Context, id string) (string, error) {
	cleanID := strings.TrimPrefix(id, "mock-")
	return fmt.Sprintf(`[{
    "Id": "%s",
    "Created": "2026-02-18T00:00:00.000000000Z",
    "Name": "/%s",
    "Path": "/docker-entrypoint.sh",
    "Args": ["-g", "daemon off;"],
    "State": {
        "Status": "running",
        "Running": true,
        "Paused": false,
        "Restarting": false,
        "OOMKilled": false,
        "Dead": false,
        "Pid": 12345,
        "ExitCode": 0,
        "StartedAt": "2026-02-18T00:00:00.000000000Z",
        "FinishedAt": "0001-01-01T00:00:00Z"
    },
    "RestartCount": 0,
    "Image": "sha256:mock-image-hash",
    "Config": {
        "Hostname": "%s",
        "Image": "mock-image:latest",
        "Cmd": ["nginx", "-g", "daemon off;"],
        "WorkingDir": "/usr/share/nginx/html",
        "User": "",
        "Env": ["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"]
    },
    "HostConfig": {
        "RestartPolicy": {
            "Name": "unless-stopped",
            "MaximumRetryCount": 0
        }
    },
    "Mounts": [
        {
            "Type": "bind",
            "Source": "/opt/stacks/data",
            "Destination": "/usr/share/nginx/html",
            "Mode": "rw",
            "RW": true
        },
        {
            "Type": "volume",
            "Name": "config-vol",
            "Source": "/var/lib/docker/volumes/config-vol/_data",
            "Destination": "/etc/nginx/conf.d",
            "Mode": "ro",
            "RW": false
        }
    ],
    "NetworkSettings": {
        "Networks": {
            "bridge": {
                "IPAddress": "172.17.0.2",
                "IPPrefixLen": 16,
                "IPv6Gateway": "",
                "GlobalIPv6Address": "",
                "GlobalIPv6PrefixLen": 0,
                "Gateway": "172.17.0.1",
                "MacAddress": "02:42:ac:11:00:02",
                "Aliases": ["nginx", "%s"]
            }
        }
    }
}]`, id, cleanID, cleanID, cleanID), nil
}

func (m *MockClient) ContainerTop(_ context.Context, id string) ([]string, [][]string, error) {
	titles := []string{"PID", "USER", "COMMAND"}
	processes := [][]string{
		{"1", "root", "nginx: master process nginx -g daemon off;"},
		{"29", "nginx", "nginx: worker process"},
		{"30", "nginx", "nginx: worker process"},
	}
	return titles, processes, nil
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

func (m *MockClient) ImageList(_ context.Context) ([]ImageSummary, error) {
	// Build container count from current state
	containers, _ := m.ContainerList(context.Background(), true, "")
	countByImage := make(map[string]int)
	for _, c := range containers {
		countByImage[c.Image]++
	}

	images := []struct {
		tag     string
		size    string
		created string
	}{
		{"nginx:latest", "187.8MiB", "2026-01-15T10:00:00Z"},
		{"nginx:1.24", "142.3MiB", "2025-09-20T08:00:00Z"},
		{"redis:7-alpine", "30.2MiB", "2026-01-10T12:00:00Z"},
		{"wordpress:6", "615.4MiB", "2026-01-18T09:00:00Z"},
		{"wordpress:6.3", "609.1MiB", "2025-08-10T14:00:00Z"},
		{"mysql:8", "573.0MiB", "2026-01-12T11:00:00Z"},
		{"grafana/grafana:latest", "402.5MiB", "2026-01-20T07:00:00Z"},
		{"alpine:3.19", "7.4MiB", "2025-12-01T06:00:00Z"},
		{"portainer/portainer-ce:latest", "295.1MiB", "2026-01-25T08:00:00Z"},
		{"containrrr/watchtower:latest", "15.2MiB", "2026-01-22T10:00:00Z"},
		{"ghcr.io/home-assistant/home-assistant:stable", "1.8GiB", "2026-01-28T06:00:00Z"},
	}

	result := make([]ImageSummary, 0, len(images)+2)
	for _, img := range images {
		hash := mockHash(img.tag)
		id := fmt.Sprintf("sha256:%s%s", hash, hash)
		result = append(result, ImageSummary{
			ID:         id,
			RepoTags:   []string{img.tag},
			Size:       img.size,
			Created:    img.created,
			Containers: countByImage[img.tag],
		})
	}

	// Dangling images (untagged)
	result = append(result, ImageSummary{
		ID:       "sha256:a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
		RepoTags: []string{},
		Size:     "245.3MiB",
		Created:  "2025-11-15T04:00:00Z",
		Dangling: true,
	})
	result = append(result, ImageSummary{
		ID:       "sha256:f6e5d4c3b2a1f6e5d4c3b2a1f6e5d4c3b2a1f6e5d4c3b2a1f6e5d4c3b2a1f6e5",
		RepoTags: []string{},
		Size:     "89.7MiB",
		Created:  "2025-10-20T02:00:00Z",
		Dangling: true,
	})

	return result, nil
}

func (m *MockClient) ImageInspectDetail(_ context.Context, imageRef string) (*ImageDetail, error) {
	// Find containers using this image
	containers, _ := m.ContainerList(context.Background(), true, "")
	var imgContainers []ImageContainer
	for _, c := range containers {
		if c.Image == imageRef {
			imgContainers = append(imgContainers, ImageContainer{
				Name:        c.Name,
				ContainerID: c.ID,
				State:       c.State,
			})
		}
	}
	if imgContainers == nil {
		imgContainers = []ImageContainer{}
	}

	hash := mockHash(imageRef)
	id := fmt.Sprintf("sha256:%s%s", hash, hash)

	type imageInfo struct {
		size       string
		created    string
		workingDir string
		layers     []ImageLayer
	}

	layerSets := map[string]imageInfo{
		"nginx:latest": {
			size: "187.8MiB", created: "2026-01-15T10:00:00Z", workingDir: "/usr/share/nginx/html",
			layers: []ImageLayer{
				{ID: id[:19], Created: "2026-01-15T10:00:00Z", Size: "0B", Command: "CMD [\"nginx\" \"-g\" \"daemon off;\"]"},
				{ID: "<missing>", Created: "2026-01-15T09:59:00Z", Size: "1.4KiB", Command: "COPY docker-entrypoint.sh / # buildkit"},
				{ID: "<missing>", Created: "2026-01-15T09:58:00Z", Size: "41.2MiB", Command: "RUN /bin/sh -c set -x && apt-get update # buildkit"},
				{ID: "<missing>", Created: "2026-01-15T09:50:00Z", Size: "146.6MiB", Command: "/bin/sh -c #(nop) ADD file:... in /"},
			},
		},
		"nginx:1.24": {
			size: "142.3MiB", created: "2025-09-20T08:00:00Z", workingDir: "/usr/share/nginx/html",
			layers: []ImageLayer{
				{ID: id[:19], Created: "2025-09-20T08:00:00Z", Size: "0B", Command: "CMD [\"nginx\" \"-g\" \"daemon off;\"]"},
				{ID: "<missing>", Created: "2025-09-20T07:59:00Z", Size: "1.4KiB", Command: "COPY docker-entrypoint.sh / # buildkit"},
				{ID: "<missing>", Created: "2025-09-20T07:50:00Z", Size: "140.9MiB", Command: "/bin/sh -c #(nop) ADD file:... in /"},
			},
		},
		"redis:7-alpine": {
			size: "30.2MiB", created: "2026-01-10T12:00:00Z", workingDir: "/data",
			layers: []ImageLayer{
				{ID: id[:19], Created: "2026-01-10T12:00:00Z", Size: "0B", Command: "CMD [\"redis-server\"]"},
				{ID: "<missing>", Created: "2026-01-10T11:58:00Z", Size: "2.1MiB", Command: "RUN /bin/sh -c addgroup -S redis # buildkit"},
				{ID: "<missing>", Created: "2026-01-10T11:50:00Z", Size: "7.8MiB", Command: "/bin/sh -c #(nop) ADD file:... in /"},
			},
		},
		"wordpress:6": {
			size: "615.4MiB", created: "2026-01-18T09:00:00Z", workingDir: "/var/www/html",
			layers: []ImageLayer{
				{ID: id[:19], Created: "2026-01-18T09:00:00Z", Size: "0B", Command: "CMD [\"apache2-foreground\"]"},
				{ID: "<missing>", Created: "2026-01-18T08:55:00Z", Size: "62.3MiB", Command: "RUN /bin/sh -c curl -o wordpress.tar.gz # buildkit"},
				{ID: "<missing>", Created: "2026-01-18T08:50:00Z", Size: "553.1MiB", Command: "/bin/sh -c #(nop) ADD file:... in /"},
			},
		},
		"wordpress:6.3": {
			size: "609.1MiB", created: "2025-08-10T14:00:00Z", workingDir: "/var/www/html",
			layers: []ImageLayer{
				{ID: id[:19], Created: "2025-08-10T14:00:00Z", Size: "0B", Command: "CMD [\"apache2-foreground\"]"},
				{ID: "<missing>", Created: "2025-08-10T13:55:00Z", Size: "60.1MiB", Command: "RUN /bin/sh -c curl -o wordpress.tar.gz # buildkit"},
				{ID: "<missing>", Created: "2025-08-10T13:50:00Z", Size: "549.0MiB", Command: "/bin/sh -c #(nop) ADD file:... in /"},
			},
		},
		"mysql:8": {
			size: "573.0MiB", created: "2026-01-12T11:00:00Z", workingDir: "",
			layers: []ImageLayer{
				{ID: id[:19], Created: "2026-01-12T11:00:00Z", Size: "0B", Command: "CMD [\"mysqld\"]"},
				{ID: "<missing>", Created: "2026-01-12T10:55:00Z", Size: "340.2MiB", Command: "RUN /bin/sh -c { echo mysql-community-server # buildkit"},
				{ID: "<missing>", Created: "2026-01-12T10:50:00Z", Size: "232.8MiB", Command: "/bin/sh -c #(nop) ADD file:... in /"},
			},
		},
		"grafana/grafana:latest": {
			size: "402.5MiB", created: "2026-01-20T07:00:00Z", workingDir: "/usr/share/grafana",
			layers: []ImageLayer{
				{ID: id[:19], Created: "2026-01-20T07:00:00Z", Size: "0B", Command: "ENTRYPOINT [\"/run.sh\"]"},
				{ID: "<missing>", Created: "2026-01-20T06:55:00Z", Size: "395.1MiB", Command: "COPY --from=build /grafana /usr/share/grafana # buildkit"},
				{ID: "<missing>", Created: "2026-01-20T06:50:00Z", Size: "7.4MiB", Command: "/bin/sh -c #(nop) ADD file:... in /"},
			},
		},
		"alpine:3.19": {
			size: "7.4MiB", created: "2025-12-01T06:00:00Z", workingDir: "",
			layers: []ImageLayer{
				{ID: id[:19], Created: "2025-12-01T06:00:00Z", Size: "0B", Command: "CMD [\"/bin/sh\"]"},
				{ID: "<missing>", Created: "2025-12-01T05:50:00Z", Size: "7.4MiB", Command: "/bin/sh -c #(nop) ADD file:... in /"},
			},
		},
		"portainer/portainer-ce:latest": {
			size: "295.1MiB", created: "2026-01-25T08:00:00Z", workingDir: "/",
			layers: []ImageLayer{
				{ID: id[:19], Created: "2026-01-25T08:00:00Z", Size: "0B", Command: "ENTRYPOINT [\"/portainer\"]"},
				{ID: "<missing>", Created: "2026-01-25T07:50:00Z", Size: "295.1MiB", Command: "/bin/sh -c #(nop) ADD file:... in /"},
			},
		},
		"containrrr/watchtower:latest": {
			size: "15.2MiB", created: "2026-01-22T10:00:00Z", workingDir: "/",
			layers: []ImageLayer{
				{ID: id[:19], Created: "2026-01-22T10:00:00Z", Size: "0B", Command: "ENTRYPOINT [\"/watchtower\"]"},
				{ID: "<missing>", Created: "2026-01-22T09:50:00Z", Size: "15.2MiB", Command: "/bin/sh -c #(nop) ADD file:... in /"},
			},
		},
		"ghcr.io/home-assistant/home-assistant:stable": {
			size: "1.8GiB", created: "2026-01-28T06:00:00Z", workingDir: "/config",
			layers: []ImageLayer{
				{ID: id[:19], Created: "2026-01-28T06:00:00Z", Size: "0B", Command: "CMD [\"python3\" \"-m\" \"homeassistant\" \"--config\" \"/config\"]"},
				{ID: "<missing>", Created: "2026-01-28T05:50:00Z", Size: "1.8GiB", Command: "/bin/sh -c #(nop) ADD file:... in /"},
			},
		},
	}

	info, ok := layerSets[imageRef]
	if !ok {
		// Handle dangling images looked up by ID
		danglingImages := map[string]imageInfo{
			"sha256:a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2": {
				size: "245.3MiB", created: "2025-11-15T04:00:00Z", workingDir: "/app",
				layers: []ImageLayer{
					{ID: "a1b2c3d4e5f6", Created: "2025-11-15T04:00:00Z", Size: "0B", Command: "CMD [\"/bin/sh\"]"},
					{ID: "<missing>", Created: "2025-11-15T03:50:00Z", Size: "245.3MiB", Command: "/bin/sh -c #(nop) ADD file:... in /"},
				},
			},
			"sha256:f6e5d4c3b2a1f6e5d4c3b2a1f6e5d4c3b2a1f6e5d4c3b2a1f6e5d4c3b2a1f6e5": {
				size: "89.7MiB", created: "2025-10-20T02:00:00Z", workingDir: "",
				layers: []ImageLayer{
					{ID: "f6e5d4c3b2a1", Created: "2025-10-20T02:00:00Z", Size: "0B", Command: "ENTRYPOINT [\"/entrypoint.sh\"]"},
					{ID: "<missing>", Created: "2025-10-20T01:50:00Z", Size: "89.7MiB", Command: "/bin/sh -c #(nop) ADD file:... in /"},
				},
			},
		}
		if dInfo, found := danglingImages[imageRef]; found {
			return &ImageDetail{
				ID:           imageRef,
				RepoTags:     []string{},
				Size:         dInfo.size,
				Created:      dInfo.created,
				Architecture: "amd64",
				OS:           "linux",
				WorkingDir:   dInfo.workingDir,
				Layers:       dInfo.layers,
				Containers:   imgContainers,
			}, nil
		}
		return nil, fmt.Errorf("image not found: %s", imageRef)
	}

	return &ImageDetail{
		ID:           id,
		RepoTags:     []string{imageRef},
		Size:         info.size,
		Created:      info.created,
		Architecture: "amd64",
		OS:           "linux",
		WorkingDir:   info.workingDir,
		Layers:       info.layers,
		Containers:   imgContainers,
	}, nil
}

func (m *MockClient) ImagePrune(_ context.Context, all bool) (string, error) {
	return "Total reclaimed space: 0B", nil
}

func (m *MockClient) NetworkList(_ context.Context) ([]NetworkSummary, error) {
	return []NetworkSummary{
		{Name: "bridge", ID: "mock-net-bridge", Driver: "bridge", Scope: "local", Containers: 4},
		{Name: "host", ID: "mock-net-host", Driver: "host", Scope: "local", Containers: 0},
		{Name: "none", ID: "mock-net-none", Driver: "null", Scope: "local", Containers: 0},
		{Name: "proxy", ID: "mock-net-proxy", Driver: "bridge", Scope: "local", Attachable: true, Containers: 2},
		{Name: "monitoring_net", ID: "mock-net-monitoring", Driver: "bridge", Scope: "local", Internal: true, Containers: 3},
		{Name: "shared-db", ID: "mock-net-shared-db", Driver: "bridge", Scope: "local", Containers: 1},
	}, nil
}

func (m *MockClient) NetworkInspect(_ context.Context, networkID string) (*NetworkDetail, error) {
	networks := map[string]*NetworkDetail{
		"bridge": {
			Name: "bridge", ID: "mock-net-bridge", Driver: "bridge", Scope: "local",
			Created: "2026-01-01T00:00:00Z",
			IPAM:    []NetworkIPAM{{Subnet: "172.17.0.0/16", Gateway: "172.17.0.1"}},
			Containers: []NetworkContainerDetail{
				{Name: "01-web-app-nginx-1", ContainerID: "mock-01-web-app-nginx-1", IPv4: "172.17.0.2/16", MAC: "02:42:ac:11:00:02", State: "running"},
				{Name: "01-web-app-redis-1", ContainerID: "mock-01-web-app-redis-1", IPv4: "172.17.0.3/16", MAC: "02:42:ac:11:00:03", State: "exited"},
				{Name: "02-blog-wordpress-1", ContainerID: "mock-02-blog-wordpress-1", IPv4: "172.17.0.4/16", MAC: "02:42:ac:11:00:04", State: "running"},
				{Name: "02-blog-mysql-1", ContainerID: "mock-02-blog-mysql-1", IPv4: "172.17.0.5/16", MAC: "02:42:ac:11:00:05", State: "running"},
			},
		},
		"host":  {Name: "host", ID: "mock-net-host", Driver: "host", Scope: "local", Created: "2026-01-01T00:00:00Z", IPAM: []NetworkIPAM{}, Containers: []NetworkContainerDetail{}},
		"none":  {Name: "none", ID: "mock-net-none", Driver: "null", Scope: "local", Created: "2026-01-01T00:00:00Z", IPAM: []NetworkIPAM{}, Containers: []NetworkContainerDetail{}},
		"proxy": {
			Name: "proxy", ID: "mock-net-proxy", Driver: "bridge", Scope: "local", Attachable: true,
			Created: "2026-01-15T00:00:00Z",
			IPAM:    []NetworkIPAM{{Subnet: "172.18.0.0/16", Gateway: "172.18.0.1"}},
			Containers: []NetworkContainerDetail{
				{Name: "01-web-app-nginx-1", ContainerID: "mock-01-web-app-nginx-1", IPv4: "172.18.0.2/16", MAC: "02:42:ac:12:00:02", State: "running"},
				{Name: "04-database-postgres-1", ContainerID: "mock-04-database-postgres-1", IPv4: "172.18.0.3/16", MAC: "02:42:ac:12:00:03", State: "running"},
			},
		},
		"monitoring_net": {
			Name: "monitoring_net", ID: "mock-net-monitoring", Driver: "bridge", Scope: "local", Internal: true,
			Created: "2026-01-10T00:00:00Z",
			IPAM:    []NetworkIPAM{{Subnet: "172.19.0.0/16", Gateway: "172.19.0.1"}},
			Containers: []NetworkContainerDetail{
				{Name: "05-multi-service-app-1", ContainerID: "mock-05-multi-service-app-1", IPv4: "172.19.0.2/16", MAC: "02:42:ac:13:00:02", State: "running"},
				{Name: "05-multi-service-api-1", ContainerID: "mock-05-multi-service-api-1", IPv4: "172.19.0.3/16", MAC: "02:42:ac:13:00:03", State: "running"},
				{Name: "05-multi-service-db-1", ContainerID: "mock-05-multi-service-db-1", IPv4: "172.19.0.4/16", MAC: "02:42:ac:13:00:04", State: "running"},
			},
		},
		"shared-db": {
			Name: "shared-db", ID: "mock-net-shared-db", Driver: "bridge", Scope: "local",
			Created: "2026-02-01T00:00:00Z",
			IPAM:    []NetworkIPAM{{Subnet: "172.20.0.0/16", Gateway: "172.20.0.1"}},
			Containers: []NetworkContainerDetail{
				{Name: "02-blog-mysql-1", ContainerID: "mock-02-blog-mysql-1", IPv4: "172.20.0.2/16", MAC: "02:42:ac:14:00:02", State: "running"},
			},
		},
	}

	// Look up by name or ID
	if detail, ok := networks[networkID]; ok {
		return detail, nil
	}
	for _, detail := range networks {
		if detail.ID == networkID {
			return detail, nil
		}
	}
	return nil, fmt.Errorf("network not found: %s", networkID)
}

func (m *MockClient) VolumeList(_ context.Context) ([]VolumeSummary, error) {
	return []VolumeSummary{
		{Name: "web-app_redis-data", Driver: "local", Mountpoint: "/var/lib/docker/volumes/web-app_redis-data/_data", Containers: 1},
		{Name: "blog_mysql-data", Driver: "local", Mountpoint: "/var/lib/docker/volumes/blog_mysql-data/_data", Containers: 1},
		{Name: "monitoring_grafana-data", Driver: "local", Mountpoint: "/var/lib/docker/volumes/monitoring_grafana-data/_data", Containers: 1},
		{Name: "database_pg-data", Driver: "local", Mountpoint: "/var/lib/docker/volumes/database_pg-data/_data", Containers: 1},
		{Name: "shared-assets", Driver: "local", Mountpoint: "/var/lib/docker/volumes/shared-assets/_data", Containers: 2},
		{Name: "backup-storage", Driver: "local", Mountpoint: "/var/lib/docker/volumes/backup-storage/_data", Containers: 0},
	}, nil
}

func (m *MockClient) VolumeInspect(_ context.Context, volumeName string) (*VolumeDetail, error) {
	volumes := map[string]*VolumeDetail{
		"web-app_redis-data": {
			Name: "web-app_redis-data", Driver: "local", Scope: "local",
			Mountpoint: "/var/lib/docker/volumes/web-app_redis-data/_data",
			Created:    "2026-01-15T10:00:00Z",
			Containers: []VolumeContainer{
				{Name: "01-web-app-redis-1", ContainerID: "mock-01-web-app-redis-1", State: "running"},
			},
		},
		"blog_mysql-data": {
			Name: "blog_mysql-data", Driver: "local", Scope: "local",
			Mountpoint: "/var/lib/docker/volumes/blog_mysql-data/_data",
			Created:    "2026-01-12T11:00:00Z",
			Containers: []VolumeContainer{
				{Name: "02-blog-mysql-1", ContainerID: "mock-02-blog-mysql-1", State: "running"},
			},
		},
		"monitoring_grafana-data": {
			Name: "monitoring_grafana-data", Driver: "local", Scope: "local",
			Mountpoint: "/var/lib/docker/volumes/monitoring_grafana-data/_data",
			Created:    "2026-01-20T07:00:00Z",
			Containers: []VolumeContainer{
				{Name: "03-monitoring-grafana-1", ContainerID: "mock-03-monitoring-grafana-1", State: "running"},
			},
		},
		"database_pg-data": {
			Name: "database_pg-data", Driver: "local", Scope: "local",
			Mountpoint: "/var/lib/docker/volumes/database_pg-data/_data",
			Created:    "2026-02-01T00:00:00Z",
			Containers: []VolumeContainer{
				{Name: "04-database-postgres-1", ContainerID: "mock-04-database-postgres-1", State: "running"},
			},
		},
		"shared-assets": {
			Name: "shared-assets", Driver: "local", Scope: "local",
			Mountpoint: "/var/lib/docker/volumes/shared-assets/_data",
			Created:    "2026-01-10T00:00:00Z",
			Containers: []VolumeContainer{
				{Name: "05-multi-service-app-1", ContainerID: "mock-05-multi-service-app-1", State: "running"},
				{Name: "05-multi-service-api-1", ContainerID: "mock-05-multi-service-api-1", State: "running"},
			},
		},
		"backup-storage": {
			Name: "backup-storage", Driver: "local", Scope: "local",
			Mountpoint: "/var/lib/docker/volumes/backup-storage/_data",
			Created:    "2025-12-01T00:00:00Z",
			Containers: []VolumeContainer{},
		},
	}

	if detail, ok := volumes[volumeName]; ok {
		return detail, nil
	}
	return nil, fmt.Errorf("volume not found: %s", volumeName)
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
