package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// MockWorld materializes the "live Docker environment" as mutable structs.
// Built once from MockData + MockState at startup. All FakeDaemon handlers
// read from this instead of independently reconstructing responses.
type MockWorld struct {
	mu         sync.RWMutex
	containers map[string]*LiveContainer  // containerID → container
	nameIndex  map[string]*LiveContainer  // container name → container (for lookup by name)
	networks   map[string]*LiveNetwork    // network name → network
	data       *MockData
	state      *MockState
	stacksDir  string
}

// LiveContainer represents a running/exited container in the mock world.
type LiveContainer struct {
	ID            string
	Name          string
	StackName     string // "" for standalone containers
	ServiceName   string
	Image         string
	ImageID       string
	Command       string
	Args          []string
	Env           []string
	WorkingDir    string
	RestartPolicy string
	State         string // "running", "exited", "paused"
	Health        string
	Mounts        []mountJSON
	Networks      map[string]*LiveEndpoint // netName → endpoint
	Ports         map[string][]portBindingJSON
	Labels        map[string]string
	Created       time.Time
	StartedAt     time.Time
	Pid           int
	IsStandalone  bool
}

// LiveEndpoint holds network endpoint data for a container.
type LiveEndpoint struct {
	NetworkID   string
	IPAddress   string
	IPPrefixLen int
	Gateway     string
	MacAddress  string
	Aliases     []string
}

// LiveNetwork represents a Docker network in the mock world.
type LiveNetwork struct {
	Name       string
	ID         string
	Driver     string
	Scope      string
	Internal   bool
	Subnet     string
	Gateway    string
	Created    string
	Containers map[string]*LiveContainer // containerID → container (back-reference)
}

// BuildMockWorld materializes the live state from MockData + MockState.
func BuildMockWorld(data *MockData, state *MockState, stacksDir string) *MockWorld {
	w := &MockWorld{
		containers: make(map[string]*LiveContainer),
		nameIndex:  make(map[string]*LiveContainer),
		networks:   make(map[string]*LiveNetwork),
		data:       data,
		state:      state,
		stacksDir:  stacksDir,
	}

	w.rebuild()
	return w
}

// rebuild reconstructs all containers and networks from data + state.
func (w *MockWorld) rebuild() {
	w.containers = make(map[string]*LiveContainer)
	w.nameIndex = make(map[string]*LiveContainer)
	w.networks = make(map[string]*LiveNetwork)

	createdTime := time.Date(2026, 2, 18, 0, 0, 0, 0, time.UTC)

	// Build networks first
	for name, meta := range w.data.networks {
		netID := w.data.NetworkID(name)
		subnet := meta.subnet
		gateway := meta.gateway
		if subnet == "" && meta.driver == "bridge" {
			// Compute deterministic subnet for networks without explicit config
			h := simpleHash(name)
			sub := 17 + int(h%200)
			subnet = fmt.Sprintf("172.%d.0.0/16", sub)
			gateway = fmt.Sprintf("172.%d.0.1", sub)
		}
		w.networks[name] = &LiveNetwork{
			Name:       name,
			ID:         netID,
			Driver:     meta.driver,
			Scope:      meta.scope,
			Internal:   meta.internal,
			Subnet:     subnet,
			Gateway:    gateway,
			Created:    "2026-01-01T00:00:00Z",
			Containers: make(map[string]*LiveContainer),
		}
	}

	// Build containers for managed stacks
	entries, _ := os.ReadDir(w.stacksDir)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		stackName := entry.Name()
		composeFile := findComposeFilePath(filepath.Join(w.stacksDir, stackName))
		if composeFile == "" {
			continue
		}

		services := w.getServiceNames(stackName)
		for _, svc := range services {
			// State will be resolved dynamically at read time;
			// use "exited" as a placeholder for structural build
			svcState := "exited"

			image := w.data.GetRunningImage(stackName, svc)
			containerID := fmt.Sprintf("mock-%s-%s-1", stackName, svc)
			containerName := fmt.Sprintf("%s-%s-1", stackName, svc)
			imageHash := mockHash(image)
			imageID := fmt.Sprintf("sha256:%s%s", imageHash, imageHash)

			health := w.data.GetServiceHealth(stackName, svc)

			key := stackName + "/" + svc

			// Command from mock.yaml, or default
			command := "/docker-entrypoint.sh"
			if cmd, ok := w.data.serviceCommands[key]; ok {
				command = cmd
			}

			// Args from mock.yaml
			var args []string
			if a, ok := w.data.serviceArgs[key]; ok {
				args = a
			}

			// Env from mock.yaml, or default
			env := []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"}
			if e, ok := w.data.serviceEnv[key]; ok {
				env = e
			}

			// Restart policy from mock.yaml, or default
			restartPolicy := "unless-stopped"
			if rp, ok := w.data.serviceRestartPolicy[key]; ok {
				restartPolicy = rp
			}

			workDir := "/usr/share/nginx/html"
			if wd := workingDirForImage(image); wd != "" {
				workDir = wd
			}

			isRunning := svcState == "running" || svcState == "paused"
			pid := 0
			if isRunning {
				pid = 12345
			}

			// Build mounts
			mounts := buildMountsFromData(w.data, stackName, svc)

			// Build port bindings
			ports := buildPortBindingsFromData(w.data, stackName, svc)

			labels := map[string]string{
				"com.docker.compose.project": stackName,
				"com.docker.compose.service": svc,
			}

			c := &LiveContainer{
				ID:            containerID,
				Name:          containerName,
				StackName:     stackName,
				ServiceName:   svc,
				Image:         image,
				ImageID:       imageID,
				Command:       command,
				Args:          args,
				Env:           env,
				WorkingDir:    workDir,
				RestartPolicy: restartPolicy,
				State:         svcState,
				Health:        health,
				Mounts:        mounts,
				Ports:         ports,
				Labels:        labels,
				Created:       createdTime,
				StartedAt:     createdTime,
				Pid:           pid,
				Networks:      make(map[string]*LiveEndpoint),
			}

			// Build network endpoints
			if eps, ok := w.data.serviceEndpoints[key]; ok && len(eps) > 0 {
				// Use explicit endpoint config from mock.yaml
				for netName, ep := range eps {
					net := w.networks[netName]
					netID := ""
					gw := ""
					if net != nil {
						netID = net.ID
						gw = net.Gateway
					} else {
						netID = w.data.NetworkID(netName)
					}
					c.Networks[netName] = &LiveEndpoint{
						NetworkID:   netID,
						IPAddress:   ep.ip,
						IPPrefixLen: 16,
						Gateway:     gw,
						MacAddress:  ep.mac,
						Aliases:     []string{svc, containerName},
					}
				}
			} else {
				// Fallback: compute endpoints from serviceNetworks
				nets, ok := w.data.serviceNetworks[key]
				if !ok || len(nets) == 0 {
					// Default to bridge
					c.Networks["bridge"] = &LiveEndpoint{
						IPAddress:   "172.17.0.2",
						IPPrefixLen: 16,
						Gateway:     "172.17.0.1",
						MacAddress:  "02:42:ac:11:00:02",
						NetworkID:   w.data.NetworkID("bridge"),
						Aliases:     []string{svc, containerName},
					}
				} else {
					for i, netName := range nets {
						subnet := 17 + i
						hostByte := 2 + simpleHash(key)%200
						c.Networks[netName] = &LiveEndpoint{
							IPAddress:   fmt.Sprintf("172.%d.0.%d", subnet, hostByte),
							IPPrefixLen: 16,
							Gateway:     fmt.Sprintf("172.%d.0.1", subnet),
							MacAddress:  fmt.Sprintf("02:42:ac:%02x:00:%02x", subnet, hostByte),
							NetworkID:   w.data.NetworkID(netName),
							Aliases:     []string{svc, containerName},
						}
					}
				}
			}

			w.containers[containerID] = c
			w.nameIndex[containerName] = c

			// Add back-references to networks
			for netName := range c.Networks {
				if net, ok := w.networks[netName]; ok {
					net.Containers[containerID] = c
				}
			}
		}
	}

	// Build containers for external stacks (always build — state resolved at read time)
	for stackName, services := range w.data.externalStacks {
		for _, svc := range services {
			// State placeholder — resolved dynamically by effectiveState
			svcState := "exited"

			image := w.data.GetRunningImage(stackName, svc)
			containerID := fmt.Sprintf("mock-%s-%s-1", stackName, svc)
			containerName := fmt.Sprintf("%s-%s-1", stackName, svc)
			imageHash := mockHash(image)
			imageID := fmt.Sprintf("sha256:%s%s", imageHash, imageHash)
			health := w.data.GetServiceHealth(stackName, svc)

			key := stackName + "/" + svc

			command := "/docker-entrypoint.sh"
			restartPolicy := "unless-stopped"
			if extSvc, ok := w.data.externalServices[key]; ok {
				if extSvc.command != "" {
					command = extSvc.command
				}
			}

			labels := map[string]string{
				"com.docker.compose.project": stackName,
				"com.docker.compose.service": svc,
			}

			workDir := ""
			if wd := workingDirForImage(image); wd != "" {
				workDir = wd
			}

			ports := buildPortBindingsFromData(w.data, stackName, svc)

			c := &LiveContainer{
				ID:            containerID,
				Name:          containerName,
				StackName:     stackName,
				ServiceName:   svc,
				Image:         image,
				ImageID:       imageID,
				Command:       command,
				Env:           []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
				WorkingDir:    workDir,
				RestartPolicy: restartPolicy,
				State:         svcState,
				Health:        health,
				Mounts:        []mountJSON{},
				Ports:         ports,
				Labels:        labels,
				Created:       createdTime,
				StartedAt:     createdTime,
				Pid:           func() int { if svcState == "running" { return 12345 }; return 0 }(),
				Networks:      make(map[string]*LiveEndpoint),
			}

			// Network endpoints for external services
			if extSvc, ok := w.data.externalServices[key]; ok && extSvc.ip != "" {
				netName := extSvc.network
				if netName == "" {
					netName = "bridge"
				}
				netID := w.data.NetworkID(netName)
				gw := "172.17.0.1"
				if net, ok := w.networks[netName]; ok {
					netID = net.ID
					if net.Gateway != "" {
						gw = net.Gateway
					}
				}
				c.Networks[netName] = &LiveEndpoint{
					NetworkID:   netID,
					IPAddress:   extSvc.ip,
					IPPrefixLen: 16,
					Gateway:     gw,
					MacAddress:  extSvc.mac,
					Aliases:     []string{svc, containerName},
				}
			} else {
				c.Networks["bridge"] = &LiveEndpoint{
					IPAddress:   "172.17.0.2",
					IPPrefixLen: 16,
					Gateway:     "172.17.0.1",
					MacAddress:  "02:42:ac:11:00:02",
					NetworkID:   w.data.NetworkID("bridge"),
					Aliases:     []string{svc, containerName},
				}
			}

			w.containers[containerID] = c
			w.nameIndex[containerName] = c

			for netName := range c.Networks {
				if net, ok := w.networks[netName]; ok {
					net.Containers[containerID] = c
				}
			}
		}
	}

	// Build standalone containers
	for _, s := range w.data.standalones {
		containerID := fmt.Sprintf("mock-standalone-%s", s.name)
		imageHash := mockHash(s.image)
		imageID := fmt.Sprintf("sha256:%s%s", imageHash, imageHash)

		command := "/entrypoint.sh"
		if s.command != "" {
			command = s.command
		}
		restartPolicy := "no"
		if s.restartPolicy != "" {
			restartPolicy = s.restartPolicy
		}

		workDir := ""
		if wd := workingDirForImage(s.image); wd != "" {
			workDir = wd
		}

		c := &LiveContainer{
			ID:            containerID,
			Name:          s.name,
			Image:         s.image,
			ImageID:       imageID,
			Command:       command,
			Env:           []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
			WorkingDir:    workDir,
			RestartPolicy: restartPolicy,
			State:         s.state,
			Mounts:        []mountJSON{},
			Labels:        map[string]string{},
			Created:       createdTime,
			StartedAt:     createdTime,
			Pid:           func() int { if s.state == "running" { return 12345 }; return 0 }(),
			IsStandalone:  true,
			Networks:      make(map[string]*LiveEndpoint),
		}

		// Network endpoint for standalone container
		netName := s.network
		if netName == "" {
			netName = "bridge"
		}
		ip := s.ip
		mac := s.mac
		if ip == "" {
			ip = "172.17.0.2"
		}
		if mac == "" {
			mac = "02:42:ac:11:00:02"
		}
		stNetID := w.data.NetworkID(netName)
		gw := "172.17.0.1"
		if net, ok := w.networks[netName]; ok {
			stNetID = net.ID
			if net.Gateway != "" {
				gw = net.Gateway
			}
		}
		c.Networks[netName] = &LiveEndpoint{
			NetworkID:   stNetID,
			IPAddress:   ip,
			IPPrefixLen: 16,
			Gateway:     gw,
			MacAddress:  mac,
			Aliases:     []string{s.name},
		}

		w.containers[containerID] = c
		w.nameIndex[c.Name] = c

		for nName := range c.Networks {
			if net, ok := w.networks[nName]; ok {
				net.Containers[containerID] = c
			}
		}
	}
}

// Reset rebuilds the world from the current MockState.
func (w *MockWorld) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.rebuild()
}

// SetStackState updates all containers in a stack to the given state.
func (w *MockWorld) SetStackState(stackName, status string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	// Rebuild after state changes - this is the simplest correct approach
	w.rebuild()
}

// RemoveStack removes all containers for a stack.
func (w *MockWorld) RemoveStack(stackName string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.rebuild()
}

// effectiveState resolves the current state of a container by reading
// from MockState at query time. This ensures state changes (e.g. from tests
// calling MockState.Set) are immediately visible without requiring a rebuild.
func (w *MockWorld) effectiveState(c *LiveContainer) string {
	if c.IsStandalone {
		return c.State
	}
	// Per-service override takes priority
	if svcState := w.state.GetService(c.StackName, c.ServiceName); svcState != "" {
		return svcState
	}
	stackStatus := w.state.Get(c.StackName)
	if stackStatus == "inactive" {
		return "exited"
	}
	// Use data-driven per-service state (e.g., some services start as exited in a running stack)
	return w.data.GetServiceState(c.StackName, c.ServiceName, stackStatus)
}

// --- Serialization methods for FakeDaemon handlers ---

// ContainerList returns containers as Docker API JSON, filtered by all/projectFilter.
func (w *MockWorld) ContainerList(all bool, projectFilter string) []containerJSON {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var result []containerJSON

	// Collect and sort container IDs for deterministic output
	ids := make([]string, 0, len(w.containers))
	for id := range w.containers {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		c := w.containers[id]

		// Filter by project
		if projectFilter != "" {
			if c.Labels["com.docker.compose.project"] != projectFilter {
				continue
			}
		}

		// Standalone containers only appear when no project filter
		if c.IsStandalone && projectFilter != "" {
			continue
		}

		// Resolve effective state dynamically from MockState
		state := w.effectiveState(c)

		// Skip inactive stacks entirely (they have no running Docker containers)
		if !c.IsStandalone {
			stackStatus := w.state.Get(c.StackName)
			if stackStatus == "inactive" && !all {
				continue
			}
		}

		// Filter non-running when all=false
		if !all && state != "running" && state != "paused" {
			continue
		}

		statusStr := buildStatusString(state, c.Health)

		// Convert networks
		nets := make(map[string]endpointJSON, len(c.Networks))
		for netName, ep := range c.Networks {
			nets[netName] = endpointJSON{
				IPAddress:   ep.IPAddress,
				IPPrefixLen: ep.IPPrefixLen,
				Gateway:     ep.Gateway,
				MacAddress:  ep.MacAddress,
				NetworkID:   ep.NetworkID,
			}
		}

		result = append(result, containerJSON{
			ID:      c.ID,
			Names:   []string{"/" + c.Name},
			Image:   c.Image,
			ImageID: c.ImageID,
			Command: c.Command,
			Created: c.Created.Unix(),
			State:   state,
			Status:  statusStr,
			Labels:  c.Labels,
			Mounts:  c.Mounts,
			NetworkSettings: &networkSettingsJSON{Networks: nets},
		})
	}

	return result
}

// ContainerInspect returns a single container's inspect response.
// Looks up by container ID first, then by container name.
func (w *MockWorld) ContainerInspect(id string) (containerInspectJSON, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	c, ok := w.containers[id]
	if !ok {
		c, ok = w.nameIndex[id]
	}
	if !ok {
		return containerInspectJSON{}, false
	}

	// Resolve effective state dynamically
	state := w.effectiveState(c)
	isRunning := state == "running" || state == "paused"
	isPaused := state == "paused"
	pid := 0
	if isRunning {
		pid = 12345
	}

	// Build inspect-style networks
	inspectNets := make(map[string]inspectEndpointJSON, len(c.Networks))
	for name, ep := range c.Networks {
		inspectNets[name] = inspectEndpointJSON{
			IPAddress:   ep.IPAddress,
			IPPrefixLen: ep.IPPrefixLen,
			Gateway:     ep.Gateway,
			MacAddress:  ep.MacAddress,
			Aliases:     ep.Aliases,
			NetworkID:   ep.NetworkID,
		}
	}

	// Build args — use container's stored args, falling back to a default
	args := c.Args
	if args == nil {
		args = []string{}
	}

	resp := containerInspectJSON{
		ID:      c.ID,
		Created: c.Created.Format("2006-01-02T15:04:05.000000000Z"),
		Name:    "/" + c.Name,
		Path:    c.Command,
		Args:    args,
		State: &containerStateJSON{
			Status:     state,
			Running:    isRunning,
			Paused:     isPaused,
			Restarting: false,
			OOMKilled:  false,
			Dead:       false,
			Pid:        pid,
			ExitCode:   0,
			StartedAt:  c.StartedAt.Format("2006-01-02T15:04:05.000000000Z"),
			FinishedAt: "0001-01-01T00:00:00Z",
		},
		RestartCount: 0,
		Image:        c.ImageID,
		Config: &containerConfigJSON{
			Hostname:   c.Name,
			Image:      c.Image,
			Cmd:        []string{c.Command},
			WorkingDir: c.WorkingDir,
			User:       "",
			Env:        c.Env,
			Tty:        false,
		},
		HostConfig: &hostConfigJSON{
			RestartPolicy: restartPolicyJSON{
				Name:              c.RestartPolicy,
				MaximumRetryCount: 0,
			},
		},
		Mounts:          c.Mounts,
		NetworkSettings: &inspectNetworkSettingsJSON{Ports: c.Ports, Networks: inspectNets},
	}

	return resp, true
}

// NetworkList returns all networks as Docker API JSON.
func (w *MockWorld) NetworkList() []networkJSON {
	w.mu.RLock()
	defer w.mu.RUnlock()

	names := make([]string, 0, len(w.networks))
	for name := range w.networks {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]networkJSON, 0, len(names))
	for _, name := range names {
		net := w.networks[name]

		var ipamConfig []ipamConfigJSON
		if net.Subnet != "" {
			ipamConfig = []ipamConfigJSON{{
				Subnet:  net.Subnet,
				Gateway: net.Gateway,
			}}
		}

		result = append(result, networkJSON{
			Name:     net.Name,
			ID:       net.ID,
			Created:  net.Created,
			Scope:    net.Scope,
			Driver:   net.Driver,
			Internal: net.Internal,
			IPAM: networkIPAMJSON{
				Driver: "default",
				Config: ipamConfig,
			},
			Containers: map[string]networkContainerJSON{},
		})
	}

	return result
}

// NetworkInspect returns a single network with its attached containers.
func (w *MockWorld) NetworkInspect(nameOrID string) (networkJSON, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var net *LiveNetwork
	for name, n := range w.networks {
		if name == nameOrID || n.ID == nameOrID {
			net = n
			break
		}
	}
	if net == nil {
		return networkJSON{}, false
	}

	// Build container list — sorted by ID for determinism
	ids := make([]string, 0, len(net.Containers))
	for id := range net.Containers {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	netContainers := make(map[string]networkContainerJSON, len(ids))
	for _, id := range ids {
		c := net.Containers[id]
		ep := c.Networks[net.Name]
		if ep == nil {
			continue
		}
		netContainers[id] = networkContainerJSON{
			Name:        c.Name,
			MacAddress:  ep.MacAddress,
			IPv4Address: ep.IPAddress + "/16",
		}
	}

	var ipamConfig []ipamConfigJSON
	if net.Subnet != "" {
		ipamConfig = []ipamConfigJSON{{
			Subnet:  net.Subnet,
			Gateway: net.Gateway,
		}}
	}

	resp := networkJSON{
		Name:       net.Name,
		ID:         net.ID,
		Created:    net.Created,
		Scope:      net.Scope,
		Driver:     net.Driver,
		Internal:   net.Internal,
		EnableIPv6: false,
		IPAM: networkIPAMJSON{
			Driver: "default",
			Config: ipamConfig,
		},
		Containers: netContainers,
	}

	return resp, true
}

// GetContainer returns a LiveContainer by ID (used by stats/logs handlers).
func (w *MockWorld) GetContainer(id string) (*LiveContainer, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	c, ok := w.containers[id]
	if !ok {
		c, ok = w.nameIndex[id]
	}
	return c, ok
}

// GetContainerState returns the effective runtime state of a container,
// resolving from MockState at query time.
func (w *MockWorld) GetContainerState(id string) string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	c, ok := w.containers[id]
	if !ok {
		c, ok = w.nameIndex[id]
	}
	if !ok {
		return "running" // unknown container defaults to running
	}
	return w.effectiveState(c)
}

// --- Helper functions ---

func (w *MockWorld) getServiceNames(stackName string) []string {
	prefix := stackName + "/"
	var services []string
	for key := range w.data.serviceImages {
		if strings.HasPrefix(key, prefix) {
			services = append(services, strings.TrimPrefix(key, prefix))
		}
	}
	sort.Strings(services)

	if len(services) == 0 {
		composeFile := findComposeFilePath(filepath.Join(w.stacksDir, stackName))
		if composeFile != "" {
			cd := parseComposeForMock(composeFile)
			for _, svc := range cd.services {
				services = append(services, svc.name)
			}
		}
	}
	return services
}

// buildMountsFromData constructs mount JSON from MockData.
func buildMountsFromData(data *MockData, stackName, svc string) []mountJSON {
	key := stackName + "/" + svc
	mounts, ok := data.serviceVolumes[key]
	if !ok {
		return []mountJSON{}
	}

	result := make([]mountJSON, 0, len(mounts))
	for _, mt := range mounts {
		rw := !mt.readOnly
		mode := "rw"
		if mt.readOnly {
			mode = "ro"
		}
		if mt.mountType == "volume" {
			result = append(result, mountJSON{
				Type:        "volume",
				Name:        mt.name,
				Source:      fmt.Sprintf("/var/lib/docker/volumes/%s/_data", mt.name),
				Destination: mt.destination,
				Mode:        mode,
				RW:          rw,
			})
		} else {
			result = append(result, mountJSON{
				Type:        "bind",
				Source:      mt.source,
				Destination: mt.destination,
				Mode:        mode,
				RW:          rw,
			})
		}
	}
	return result
}

// buildPortBindingsFromData constructs port bindings from MockData.
func buildPortBindingsFromData(data *MockData, stackName, svc string) map[string][]portBindingJSON {
	key := stackName + "/" + svc
	ports, ok := data.servicePorts[key]
	if !ok || len(ports) == 0 {
		return nil
	}

	result := make(map[string][]portBindingJSON)
	for _, p := range ports {
		proto := "tcp"
		if idx := strings.LastIndex(p, "/"); idx >= 0 {
			proto = p[idx+1:]
			p = p[:idx]
		}

		parts := strings.Split(p, ":")
		var hostIP, hostPort, containerPort string
		switch len(parts) {
		case 1:
			containerPort = parts[0]
			hostPort = parts[0]
		case 2:
			hostPort = parts[0]
			containerPort = parts[1]
		case 3:
			hostIP = parts[0]
			hostPort = parts[1]
			containerPort = parts[2]
		}
		if hostIP == "" {
			hostIP = "0.0.0.0"
		}

		key := containerPort + "/" + proto
		result[key] = append(result[key], portBindingJSON{
			HostIp:   hostIP,
			HostPort: hostPort,
		})
	}
	return result
}
