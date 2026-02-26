package docker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
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
	data      *MockData
}

// NewMockClient returns a new in-memory MockClient.
// If no MockData is provided, it builds one from the stacks directory.
func NewMockClient(stacksDir string, state *MockState) *MockClient {
	return &MockClient{
		stacksDir: stacksDir,
		state:     state,
		data:      BuildMockData(stacksDir),
	}
}

// NewMockClientWithData returns a MockClient using pre-built MockData.
func NewMockClientWithData(stacksDir string, state *MockState, data *MockData) *MockClient {
	return &MockClient{
		stacksDir: stacksDir,
		state:     state,
		data:      data,
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
			svcState := m.data.GetServiceState(stackName, svc, status)

			// Skip stopped containers unless all=true
			if !all && svcState != "running" {
				continue
			}

			image := m.data.GetRunningImage(stackName, svc)
			containers = append(containers, Container{
				ID:      fmt.Sprintf("mock-%s-%s-1", stackName, svc),
				Name:    fmt.Sprintf("%s-%s-1", stackName, svc),
				Project: stackName,
				Service: svc,
				Image:   image,
				State:   svcState,
				Health:  m.data.GetServiceHealth(stackName, svc),
			})
		}
	}

	// Append standalone containers (no compose project) when not filtering by project
	if projectFilter == "" {
		for _, s := range m.data.standalones {
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

	// Determine the actual image for this container
	image := "mock-image:latest"
	workDir := "/usr/share/nginx/html"
	stack, svc, ok := m.data.parseContainerKey(id)
	if ok {
		image = m.data.GetRunningImage(stack, svc)
		if wd := workingDirForImage(image); wd != "" {
			workDir = wd
		}
	}

	imageHash := mockHash(image)

	// Build mounts JSON
	mountsJSON := m.buildMountsJSON(stack, svc)

	// Build networks JSON
	networksJSON := m.buildNetworksJSON(stack, svc, cleanID)

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
    "Image": "sha256:%s%s",
    "Config": {
        "Hostname": "%s",
        "Image": "%s",
        "Cmd": ["nginx", "-g", "daemon off;"],
        "WorkingDir": "%s",
        "User": "",
        "Env": ["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"]
    },
    "HostConfig": {
        "RestartPolicy": {
            "Name": "unless-stopped",
            "MaximumRetryCount": 0
        }
    },
    "Mounts": %s,
    "NetworkSettings": {
        "Networks": %s
    }
}]`, id, cleanID, imageHash, imageHash, cleanID, image, workDir, mountsJSON, networksJSON), nil
}

func (m *MockClient) buildMountsJSON(stack, svc string) string {
	key := stack + "/" + svc
	mounts, ok := m.data.serviceVolumes[key]
	if !ok || len(mounts) == 0 {
		return "[]"
	}

	var parts []string
	for _, mt := range mounts {
		rw := "true"
		mode := "rw"
		if mt.readOnly {
			rw = "false"
			mode = "ro"
		}
		if mt.mountType == "volume" {
			parts = append(parts, fmt.Sprintf(`{
            "Type": "volume",
            "Name": "%s",
            "Source": "/var/lib/docker/volumes/%s/_data",
            "Destination": "%s",
            "Mode": "%s",
            "RW": %s
        }`, mt.name, mt.name, mt.destination, mode, rw))
		} else {
			parts = append(parts, fmt.Sprintf(`{
            "Type": "bind",
            "Source": "%s",
            "Destination": "%s",
            "Mode": "%s",
            "RW": %s
        }`, mt.source, mt.destination, mode, rw))
		}
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func (m *MockClient) buildNetworksJSON(stack, svc, cleanID string) string {
	key := stack + "/" + svc
	nets, ok := m.data.serviceNetworks[key]
	if !ok || len(nets) == 0 {
		// Default: bridge network
		return fmt.Sprintf(`{
            "bridge": {
                "IPAddress": "172.17.0.2",
                "IPPrefixLen": 16,
                "IPv6Gateway": "",
                "GlobalIPv6Address": "",
                "GlobalIPv6PrefixLen": 0,
                "Gateway": "172.17.0.1",
                "MacAddress": "02:42:ac:11:00:02",
                "Aliases": ["%s"]
            }
        }`, cleanID)
	}

	var parts []string
	for i, netName := range nets {
		subnet := 17 + i
		hostByte := 2 + simpleHash(key)%200
		parts = append(parts, fmt.Sprintf(`"%s": {
                "IPAddress": "172.%d.0.%d",
                "IPPrefixLen": 16,
                "IPv6Gateway": "",
                "GlobalIPv6Address": "",
                "GlobalIPv6PrefixLen": 0,
                "Gateway": "172.%d.0.1",
                "MacAddress": "02:42:ac:%02x:00:%02x",
                "Aliases": ["%s", "%s"]
            }`, netName, subnet, hostByte, subnet, subnet, hostByte, svc, cleanID))
	}
	return "{" + strings.Join(parts, ", ") + "}"
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

func (m *MockClient) ContainerStartedAt(_ context.Context, _ string) (time.Time, error) {
	return time.Time{}, nil
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

	// If any service using this image has update_available, return a different digest
	var hash string
	if m.data.HasUpdateAvailable(imageRef) {
		hash = mockHash(repo + ":" + tag + ":remote-newer")
	} else {
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

	// Collect all non-dangling images, sorted for stable output
	refs := m.data.SortedImages()

	result := make([]ImageSummary, 0, len(refs)+2)
	for _, ref := range refs {
		meta := m.data.images[ref]
		hash := mockHash(ref)
		id := fmt.Sprintf("sha256:%s%s", hash, hash)
		result = append(result, ImageSummary{
			ID:         id,
			RepoTags:   []string{ref},
			Size:       meta.size,
			Created:    meta.created,
			Containers: countByImage[ref],
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

	// Check dangling images first
	danglingIDs := map[string]imageMeta{
		"sha256:a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2": {size: "245.3MiB", created: "2025-11-15T04:00:00Z"},
		"sha256:f6e5d4c3b2a1f6e5d4c3b2a1f6e5d4c3b2a1f6e5d4c3b2a1f6e5d4c3b2a1f6e5": {size: "89.7MiB", created: "2025-10-20T02:00:00Z"},
	}
	if dInfo, found := danglingIDs[imageRef]; found {
		return &ImageDetail{
			ID:           imageRef,
			RepoTags:     []string{},
			Size:         dInfo.size,
			Created:      dInfo.created,
			Architecture: "amd64",
			OS:           "linux",
			WorkingDir:   "",
			Layers:       generateLayers(imageRef, imageRef[:12], dInfo.created),
			Containers:   imgContainers,
		}, nil
	}

	// Look up from MockData
	meta, ok := m.data.images[imageRef]
	if !ok {
		return nil, fmt.Errorf("image not found: %s", imageRef)
	}

	wd := workingDirForImage(imageRef)

	return &ImageDetail{
		ID:           id,
		RepoTags:     []string{imageRef},
		Size:         meta.size,
		Created:      meta.created,
		Architecture: "amd64",
		OS:           "linux",
		WorkingDir:   wd,
		Layers:       generateLayers(imageRef, id[:19], meta.created),
		Containers:   imgContainers,
	}, nil
}

// generateLayers creates 2-4 deterministic layers for an image.
func generateLayers(imageRef, topID, created string) []ImageLayer {
	h := simpleHash(imageRef)
	numLayers := 2 + int(h%3) // 2-4 layers

	layers := make([]ImageLayer, 0, numLayers)

	// Top layer — always the CMD/ENTRYPOINT
	cmd := fmt.Sprintf("CMD [\"/bin/sh\"]")
	baseName := imageRef
	if idx := strings.LastIndex(baseName, "/"); idx >= 0 {
		baseName = baseName[idx+1:]
	}
	if idx := strings.Index(baseName, ":"); idx >= 0 {
		baseName = baseName[:idx]
	}
	// Customize CMD for known images
	switch baseName {
	case "nginx", "httpd":
		cmd = `CMD ["nginx", "-g", "daemon off;"]`
	case "redis":
		cmd = `CMD ["redis-server"]`
	case "postgres":
		cmd = `CMD ["postgres"]`
	case "mysql", "mariadb":
		cmd = `CMD ["mysqld"]`
	case "node":
		cmd = `CMD ["node"]`
	case "python":
		cmd = `CMD ["python3"]`
	case "grafana":
		cmd = `ENTRYPOINT ["/run.sh"]`
	case "wordpress":
		cmd = `CMD ["apache2-foreground"]`
	case "traefik":
		cmd = `ENTRYPOINT ["/entrypoint.sh"]`
	case "elasticsearch":
		cmd = `ENTRYPOINT ["/bin/tini", "--", "/usr/local/bin/docker-entrypoint.sh"]`
	case "rabbitmq":
		cmd = `CMD ["rabbitmq-server"]`
	}

	layers = append(layers, ImageLayer{
		ID:      topID,
		Created: created,
		Size:    "0B",
		Command: cmd,
	})

	// Middle layers (if any)
	for i := 1; i < numLayers-1; i++ {
		layerSize := fmt.Sprintf("%.1fMiB", float64(1+h%200)+float64(i)*10)
		layers = append(layers, ImageLayer{
			ID:      "<missing>",
			Created: created,
			Size:    layerSize,
			Command: fmt.Sprintf("RUN /bin/sh -c set -x && install dependencies # buildkit"),
		})
	}

	// Base layer — always an ADD
	baseSize := fmt.Sprintf("%.1fMiB", float64(5+h%500))
	layers = append(layers, ImageLayer{
		ID:      "<missing>",
		Created: created,
		Size:    baseSize,
		Command: "/bin/sh -c #(nop) ADD file:... in /",
	})

	return layers
}

func (m *MockClient) ImagePrune(_ context.Context, all bool) (string, error) {
	return "Total reclaimed space: 0B", nil
}

func (m *MockClient) NetworkList(_ context.Context) ([]NetworkSummary, error) {
	// Build container counts dynamically
	containers, _ := m.ContainerList(context.Background(), true, "")

	// Map container → networks
	netContainerCount := make(map[string]int)
	for _, c := range containers {
		if c.State != "running" {
			continue
		}
		key := c.Project + "/" + c.Service
		if nets, ok := m.data.serviceNetworks[key]; ok {
			for _, n := range nets {
				netContainerCount[n]++
			}
		} else {
			// No explicit networks — count toward bridge
			netContainerCount["bridge"]++
		}
	}

	names := m.data.SortedNetworks()
	result := make([]NetworkSummary, 0, len(names))
	for _, name := range names {
		meta := m.data.networks[name]
		result = append(result, NetworkSummary{
			Name:       name,
			ID:         fmt.Sprintf("mock-net-%s", strings.ReplaceAll(name, "_", "-")),
			Driver:     meta.driver,
			Scope:      meta.scope,
			Internal:   meta.internal,
			Containers: netContainerCount[name],
		})
	}

	return result, nil
}

func (m *MockClient) NetworkInspect(_ context.Context, networkID string) (*NetworkDetail, error) {
	// Find network by name or ID
	var netName string
	var meta networkMeta
	found := false

	for name, nm := range m.data.networks {
		expectedID := fmt.Sprintf("mock-net-%s", strings.ReplaceAll(name, "_", "-"))
		if name == networkID || expectedID == networkID {
			netName = name
			meta = nm
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("network not found: %s", networkID)
	}

	netID := fmt.Sprintf("mock-net-%s", strings.ReplaceAll(netName, "_", "-"))

	// Find containers on this network
	containers, _ := m.ContainerList(context.Background(), true, "")
	var netContainers []NetworkContainerDetail

	subnet := 17
	// Assign a deterministic subnet index based on network name
	h := simpleHash(netName)
	subnet = 17 + int(h%200)

	ipCounter := 2
	for _, c := range containers {
		key := c.Project + "/" + c.Service
		nets, hasExplicit := m.data.serviceNetworks[key]

		onNetwork := false
		if hasExplicit {
			for _, n := range nets {
				if n == netName {
					onNetwork = true
					break
				}
			}
		} else if netName == "bridge" || netName == c.Project+"_default" {
			onNetwork = true
		}

		if onNetwork {
			netContainers = append(netContainers, NetworkContainerDetail{
				Name:        c.Name,
				ContainerID: c.ID,
				IPv4:        fmt.Sprintf("172.%d.0.%d/16", subnet, ipCounter),
				MAC:         fmt.Sprintf("02:42:ac:%02x:00:%02x", subnet%256, ipCounter),
				State:       c.State,
			})
			ipCounter++
		}
	}

	if netContainers == nil {
		netContainers = []NetworkContainerDetail{}
	}

	// IPAM config
	var ipam []NetworkIPAM
	if meta.driver == "bridge" {
		ipam = []NetworkIPAM{{
			Subnet:  fmt.Sprintf("172.%d.0.0/16", subnet),
			Gateway: fmt.Sprintf("172.%d.0.1", subnet),
		}}
	} else {
		ipam = []NetworkIPAM{}
	}

	return &NetworkDetail{
		Name:       netName,
		ID:         netID,
		Driver:     meta.driver,
		Scope:      meta.scope,
		Internal:   meta.internal,
		Created:    "2026-01-01T00:00:00Z",
		IPAM:       ipam,
		Containers: netContainers,
	}, nil
}

func (m *MockClient) VolumeList(_ context.Context) ([]VolumeSummary, error) {
	// Build container counts dynamically
	containers, _ := m.ContainerList(context.Background(), true, "")

	volContainerCount := make(map[string]int)
	for _, c := range containers {
		key := c.Project + "/" + c.Service
		if mounts, ok := m.data.serviceVolumes[key]; ok {
			for _, mt := range mounts {
				if mt.mountType == "volume" && mt.name != "" {
					volContainerCount[mt.name]++
				}
			}
		}
	}

	names := m.data.SortedVolumes()
	result := make([]VolumeSummary, 0, len(names))
	for _, name := range names {
		result = append(result, VolumeSummary{
			Name:       name,
			Driver:     "local",
			Mountpoint: fmt.Sprintf("/var/lib/docker/volumes/%s/_data", name),
			Containers: volContainerCount[name],
		})
	}

	return result, nil
}

func (m *MockClient) VolumeInspect(_ context.Context, volumeName string) (*VolumeDetail, error) {
	if _, ok := m.data.volumes[volumeName]; !ok {
		return nil, fmt.Errorf("volume not found: %s", volumeName)
	}

	// Find containers using this volume
	containers, _ := m.ContainerList(context.Background(), true, "")
	var volContainers []VolumeContainer

	for _, c := range containers {
		key := c.Project + "/" + c.Service
		if mounts, ok := m.data.serviceVolumes[key]; ok {
			for _, mt := range mounts {
				if mt.mountType == "volume" && mt.name == volumeName {
					volContainers = append(volContainers, VolumeContainer{
						Name:        c.Name,
						ContainerID: c.ID,
						State:       c.State,
					})
					break
				}
			}
		}
	}

	if volContainers == nil {
		volContainers = []VolumeContainer{}
	}

	return &VolumeDetail{
		Name:       volumeName,
		Driver:     "local",
		Scope:      "local",
		Mountpoint: fmt.Sprintf("/var/lib/docker/volumes/%s/_data", volumeName),
		Created:    "2026-01-01T00:00:00Z",
		Containers: volContainers,
	}, nil
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
	// Use MockData's serviceImages to find services for this stack
	var services []string
	prefix := stackName + "/"
	for key := range m.data.serviceImages {
		if strings.HasPrefix(key, prefix) {
			svc := strings.TrimPrefix(key, prefix)
			services = append(services, svc)
		}
	}
	sort.Strings(services)

	// If MockData has no services (e.g., stack was added after BuildMockData),
	// fall back to parsing the file directly.
	if len(services) == 0 {
		return m.parseServicesFromFile(composeFile)
	}
	return services
}

func (m *MockClient) parseServicesFromFile(composeFile string) []string {
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
