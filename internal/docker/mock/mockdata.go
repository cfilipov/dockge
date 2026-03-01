package mock

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// --- YAML unmarshaling types (exported fields required by yaml.v3) ---

// mockYAMLFile is the schema for per-stack mock.yaml files.
type mockYAMLFile struct {
	Status   string                          `yaml:"status"`
	Services map[string]serviceYAMLOverrides `yaml:"services"`
	Networks map[string]networkYAMLConfig    `yaml:"networks"`
}

// networkYAMLConfig holds per-stack network config from mock.yaml.
type networkYAMLConfig struct {
	ID string `yaml:"id"`
}

type serviceYAMLOverrides struct {
	State           string                               `yaml:"state"`
	Health          string                               `yaml:"health"`
	RunningImage    string                               `yaml:"running_image"`
	UpdateAvailable bool                                 `yaml:"update_available"`
	Command         string                               `yaml:"command"`
	Args            []string                             `yaml:"args"`
	Env             []string                             `yaml:"env"`
	RestartPolicy   string                               `yaml:"restart_policy"`
	Networks        map[string]networkEndpointYAMLConfig `yaml:"networks"`
	Logs            serviceLogsYAML                      `yaml:"logs"`
}

// serviceLogsYAML is the YAML schema for per-service log definitions.
type serviceLogsYAML struct {
	BaseTime  string        `yaml:"base_time"`
	Startup   []string      `yaml:"startup"`
	Heartbeat heartbeatYAML `yaml:"heartbeat"`
	Shutdown  []string      `yaml:"shutdown"`
}

type heartbeatYAML struct {
	Interval string   `yaml:"interval"`
	Lines    []string `yaml:"lines"`
}

// ServiceLogs is the resolved, ready-to-use log definition for a service.
type ServiceLogs struct {
	BaseTime  time.Time
	Startup   []string
	Heartbeat []string
	Interval  time.Duration
	Shutdown  []string
}

type networkEndpointYAMLConfig struct {
	IP  string `yaml:"ip"`
	MAC string `yaml:"mac"`
}

// globalMockYAMLFile is the schema for the root-level mock.yaml.
type globalMockYAMLFile struct {
	Networks             map[string]networkDefYAML         `yaml:"networks"`
	StandaloneContainers []standaloneContainerYAML         `yaml:"standalone_containers"`
	ExternalStacks       map[string]externalStackYAML      `yaml:"external_stacks"`
	DanglingImages       []danglingImageYAML               `yaml:"dangling_images"`
}

type networkDefYAML struct {
	Driver   string `yaml:"driver"`
	Internal bool   `yaml:"internal"`
	Subnet   string `yaml:"subnet"`
	Gateway  string `yaml:"gateway"`
	ID       string `yaml:"id"`
}

type standaloneContainerYAML struct {
	Name          string `yaml:"name"`
	Image         string `yaml:"image"`
	State         string `yaml:"state"`
	Command       string `yaml:"command"`
	RestartPolicy string `yaml:"restart_policy"`
	Network       string `yaml:"network"`
	IP            string `yaml:"ip"`
	MAC           string `yaml:"mac"`
}

type externalStackYAML struct {
	Services map[string]externalServiceYAML `yaml:"services"`
}

type externalServiceYAML struct {
	Image   string   `yaml:"image"`
	Command string   `yaml:"command"`
	Ports   []string `yaml:"ports"`
	Network string   `yaml:"network"`
	IP      string   `yaml:"ip"`
	MAC     string   `yaml:"mac"`
}

type danglingImageYAML struct {
	ID      string `yaml:"id"`
	Size    string `yaml:"size"`
	Created string `yaml:"created"`
}

// MockData holds all data derived from compose.yaml and mock.yaml files
// on disk. It is built once at startup and used by the fake daemon to serve
// realistic, consistent responses without hardcoded conditionals.
type MockData struct {
	// Derived from compose.yaml files
	images   map[string]imageMeta  // "nginx:latest" → size/created
	networks map[string]networkMeta // full Docker name → driver/scope
	volumes  map[string]volumeMeta  // full Docker name → project

	// Per-service data derived from compose.yaml
	serviceImages   map[string]string       // "stackName/svc" → compose image
	serviceVolumes  map[string][]volumeMount // "stackName/svc" → mounts
	serviceNetworks map[string][]string     // "stackName/svc" → full network names
	servicePorts    map[string][]string     // "stackName/svc" → port mappings

	// Derived from mock.yaml files
	stackStatuses map[string]string // stack → initial status
	serviceStates map[string]string // "stackName/svc" → "exited"
	serviceHealth map[string]string // "stackName/svc" → "unhealthy"
	runningImages map[string]string // "stackName/svc" → override image
	updateFlags   map[string]bool   // "stackName/svc" → update available

	// Per-service config from mock.yaml (command, args, env, restart policy)
	serviceCommands       map[string]string   // "stackName/svc" → command
	serviceArgs           map[string][]string // "stackName/svc" → args
	serviceEnv            map[string][]string // "stackName/svc" → env vars
	serviceRestartPolicy  map[string]string   // "stackName/svc" → restart policy
	serviceEndpoints      map[string]map[string]endpointConfig // "stackName/svc" → netName → {ip, mac}

	// Log templates and per-service resolved logs
	logTemplates map[string]*ServiceLogs // image base → default logs
	serviceLogs  map[string]*ServiceLogs // "stack/svc" → resolved logs

	// Standalone containers (not part of any compose project)
	standalones []standaloneContainer

	// External stacks (have Docker containers but no compose file in stacks dir)
	externalStacks   map[string][]string // stackName → service names
	externalServices map[string]externalServiceConfig // "stackName/svc" → config

	// Dangling images
	danglingImages []danglingImageConfig
}

type imageMeta struct {
	size    string
	created string
}

type networkMeta struct {
	driver   string
	scope    string
	internal bool
	project  string // empty for docker defaults
	subnet   string // e.g. "172.18.0.0/16"
	gateway  string // e.g. "172.18.0.1"
	id       string // 64-char hex hash (from mock.yaml)
}

type volumeMeta struct {
	project string
}

type volumeMount struct {
	name        string // volume name (for named volumes)
	source      string // source path (for binds)
	destination string
	mountType   string // "volume" or "bind"
	readOnly    bool
}

type standaloneContainer struct {
	name          string
	image         string
	state         string
	command       string
	restartPolicy string
	network       string
	ip            string
	mac           string
}

// composeService holds data extracted from a single service in compose.yaml.
type composeService struct {
	name     string
	image    string
	networks []string // network names as declared in the service
	volumes  []composeVolumeRef
	ports    []string // port mappings like "3000:3000", "8080:80"
}

type composeVolumeRef struct {
	name        string // volume name or host path
	destination string
	readOnly    bool
	isNamed     bool // true if references a top-level named volume
}

// composeData holds data extracted from a full compose.yaml file.
type composeData struct {
	services []composeService
	networks []string // top-level network names
	volumes  []string // top-level named volume names
}

// globalMockConfig holds data parsed from the root-level mock.yaml file
// in the stacks directory. Defines Docker resources that exist independently
// of any compose project.
type globalMockConfig struct {
	networks    map[string]networkMeta                  // standalone network name → metadata
	standalones []standaloneContainer                   // standalone containers
	externals   map[string]map[string]externalServiceConfig // stackName → svc → config
	danglings   []danglingImageConfig                   // dangling images
}

// mockOverrides holds data parsed from a per-stack mock.yaml sidecar file.
type mockOverrides struct {
	status   string                      // stack-level status
	services map[string]serviceOverrides // per-service overrides
	networks map[string]string           // network name → 64-char hex ID
}

// serviceOverrides holds per-service overrides from mock.yaml.
type serviceOverrides struct {
	state           string // "running", "exited"
	health          string // "", "unhealthy", "healthy"
	runningImage    string // override the running image
	updateAvailable bool   // simulate registry update
	command         string // override entrypoint command
	args            []string
	env             []string
	restartPolicy   string // "always", "unless-stopped", etc.
	networks        map[string]endpointConfig // netName → {ip, mac}
	logs            serviceLogsYAML           // per-service log overrides
}

// endpointConfig holds network endpoint config from mock.yaml.
type endpointConfig struct {
	ip  string
	mac string
}

// danglingImageConfig holds dangling image data from mock.yaml.
type danglingImageConfig struct {
	id      string
	size    string
	created string
}

// externalServiceConfig holds per-service config for external stacks.
type externalServiceConfig struct {
	image   string
	command string
	ports   []string
	network string
	ip      string
	mac     string
}

// BuildMockData scans the stacks directory, parses compose.yaml and mock.yaml
// files, and returns a fully populated MockData.
func BuildMockData(stacksDir string) *MockData {
	d := &MockData{
		images:          make(map[string]imageMeta),
		networks:        make(map[string]networkMeta),
		volumes:         make(map[string]volumeMeta),
		serviceImages:   make(map[string]string),
		serviceVolumes:  make(map[string][]volumeMount),
		serviceNetworks: make(map[string][]string),
		servicePorts:    make(map[string][]string),
		stackStatuses:   make(map[string]string),
		serviceStates:   make(map[string]string),
		serviceHealth:   make(map[string]string),
		runningImages:   make(map[string]string),
		updateFlags:     make(map[string]bool),
		serviceCommands:      make(map[string]string),
		serviceArgs:          make(map[string][]string),
		serviceEnv:           make(map[string][]string),
		serviceRestartPolicy: make(map[string]string),
		serviceEndpoints:     make(map[string]map[string]endpointConfig),
		externalServices:     make(map[string]externalServiceConfig),
		logTemplates:         make(map[string]*ServiceLogs),
		serviceLogs:          make(map[string]*ServiceLogs),
	}

	// Docker default networks (fallback when global mock.yaml doesn't define them)
	d.networks["bridge"] = networkMeta{driver: "bridge", scope: "local"}
	d.networks["host"] = networkMeta{driver: "host", scope: "local"}
	d.networks["none"] = networkMeta{driver: "null", scope: "local"}

	// Global mock config (root-level mock.yaml in stacks dir)
	// Overrides defaults with richer metadata (IDs, subnets, etc.)
	globalCfg := parseGlobalMockYAML(filepath.Join(stacksDir, "mock.yaml"))
	for name, meta := range globalCfg.networks {
		d.networks[name] = meta
	}

	// Standalone containers (from global mock.yaml, with defaults as fallback)
	if len(globalCfg.standalones) > 0 {
		d.standalones = globalCfg.standalones
	} else {
		d.standalones = []standaloneContainer{
			{name: "portainer", image: "portainer/portainer-ce:latest", state: "running"},
			{name: "watchtower", image: "containrrr/watchtower:latest", state: "running"},
			{name: "homeassistant", image: "ghcr.io/home-assistant/home-assistant:stable", state: "exited"},
		}
	}
	for _, s := range d.standalones {
		d.addImage(s.image)
	}

	// External stacks (from global mock.yaml, with defaults as fallback)
	if len(globalCfg.externals) > 0 {
		d.externalStacks = make(map[string][]string, len(globalCfg.externals))
		for stackName, svcs := range globalCfg.externals {
			var svcNames []string
			for svcName, svc := range svcs {
				svcNames = append(svcNames, svcName)
				key := stackName + "/" + svcName
				d.serviceImages[key] = svc.image
				if len(svc.ports) > 0 {
					d.servicePorts[key] = svc.ports
				}
				d.externalServices[key] = svc
				d.addImage(svc.image)
			}
			d.externalStacks[stackName] = svcNames
		}
	} else {
		d.externalStacks = map[string][]string{
			"10-unmanaged": {"web", "cache"},
		}
		d.serviceImages["10-unmanaged/web"] = "nginx:1.25"
		d.serviceImages["10-unmanaged/cache"] = "redis:7-alpine"
		d.servicePorts["10-unmanaged/web"] = []string{"8080:80", "8443:443"}
		d.addImage("nginx:1.25")
		d.addImage("redis:7-alpine")
	}

	// External stacks default to "running" status
	for stackName := range d.externalStacks {
		if _, exists := d.stackStatuses[stackName]; !exists {
			d.stackStatuses[stackName] = "running"
		}
	}

	// Dangling images (from global mock.yaml, with defaults as fallback)
	if len(globalCfg.danglings) > 0 {
		d.danglingImages = globalCfg.danglings
	} else {
		d.danglingImages = []danglingImageConfig{
			{id: "sha256:abc123dead", size: "245.3MiB", created: "2025-11-15T04:00:00Z"},
			{id: "sha256:def456beef", size: "89.7MiB", created: "2025-10-20T02:00:00Z"},
		}
	}

	// Parse log templates (log-templates.yaml in stacks dir)
	d.logTemplates = parseLogTemplates(filepath.Join(stacksDir, "log-templates.yaml"))

	entries, err := os.ReadDir(stacksDir)
	if err != nil {
		return d
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		stackName := entry.Name()
		stackDir := filepath.Join(stacksDir, stackName)

		// Parse compose.yaml
		composeFile := findComposeFilePath(stackDir)
		if composeFile == "" {
			continue
		}

		cd := parseComposeForMock(composeFile)

		// Register images and service data
		for _, svc := range cd.services {
			key := stackName + "/" + svc.name
			img := svc.image
			if img == "" {
				img = "mock-image:latest"
			}
			d.serviceImages[key] = img
			d.addImage(img)

			// Service networks
			if len(svc.networks) > 0 {
				var fullNames []string
				for _, n := range svc.networks {
					fullName := stackName + "_" + n
					fullNames = append(fullNames, fullName)
				}
				d.serviceNetworks[key] = fullNames
			} else if len(cd.networks) == 0 {
				// Default network for stacks without explicit networks
				defaultNet := stackName + "_default"
				d.serviceNetworks[key] = []string{defaultNet}
			}

			// Service volumes
			var mounts []volumeMount
			for _, v := range svc.volumes {
				if v.isNamed {
					fullName := stackName + "_" + v.name
					mounts = append(mounts, volumeMount{
						name:        fullName,
						destination: v.destination,
						mountType:   "volume",
						readOnly:    v.readOnly,
					})
				} else if v.name != "" {
					mounts = append(mounts, volumeMount{
						source:      v.name,
						destination: v.destination,
						mountType:   "bind",
						readOnly:    v.readOnly,
					})
				}
			}
			if len(mounts) > 0 {
				d.serviceVolumes[key] = mounts
			}

			// Service ports
			if len(svc.ports) > 0 {
				d.servicePorts[key] = svc.ports
			}
		}

		// Register top-level networks
		if len(cd.networks) > 0 {
			for _, n := range cd.networks {
				fullName := stackName + "_" + n
				d.networks[fullName] = networkMeta{driver: "bridge", scope: "local", project: stackName}
			}
		} else if len(cd.services) > 0 {
			// Stacks without explicit networks get a default network
			defaultNet := stackName + "_default"
			if _, exists := d.networks[defaultNet]; !exists {
				d.networks[defaultNet] = networkMeta{driver: "bridge", scope: "local", project: stackName}
			}
		}

		// Register top-level volumes
		for _, v := range cd.volumes {
			fullName := stackName + "_" + v
			d.volumes[fullName] = volumeMeta{project: stackName}
		}

		// Parse mock.yaml overrides
		mockFile := filepath.Join(stackDir, "mock.yaml")
		overrides := parseMockYAML(mockFile)

		if overrides.status != "" {
			d.stackStatuses[stackName] = overrides.status
		}

		for svcName, so := range overrides.services {
			key := stackName + "/" + svcName
			if so.state != "" {
				d.serviceStates[key] = so.state
			}
			if so.health != "" {
				d.serviceHealth[key] = so.health
			}
			if so.runningImage != "" {
				d.runningImages[key] = so.runningImage
				d.addImage(so.runningImage)
			}
			if so.updateAvailable {
				d.updateFlags[key] = true
			}
			if so.command != "" {
				d.serviceCommands[key] = so.command
			}
			if len(so.args) > 0 {
				d.serviceArgs[key] = so.args
			}
			if len(so.env) > 0 {
				d.serviceEnv[key] = so.env
			}
			if so.restartPolicy != "" {
				d.serviceRestartPolicy[key] = so.restartPolicy
			}
			if len(so.networks) > 0 {
				d.serviceEndpoints[key] = so.networks
			}
			if so.logs.hasContent() {
				d.serviceLogs[key] = so.logs.resolve()
			}
		}

		// Resolve logs for all services in this stack
		for _, svc := range cd.services {
			key := stackName + "/" + svc.name
			if _, ok := d.serviceLogs[key]; ok {
				continue // already has per-service override
			}
			// Look up image-based template
			img := svc.image
			if img == "" {
				img = "mock-image:latest"
			}
			imageBase := extractImageBaseName(img)
			if tmpl, ok := d.logTemplates[imageBase]; ok {
				d.serviceLogs[key] = tmpl
			}
		}

		// Propagate network IDs from per-stack mock.yaml
		for netName, netID := range overrides.networks {
			if meta, ok := d.networks[netName]; ok {
				meta.id = netID
				d.networks[netName] = meta
			}
		}
	}

	// Add dangling images from global mock.yaml
	for i, di := range d.danglingImages {
		key := fmt.Sprintf("<dangling:%d>", i+1)
		d.images[key] = imageMeta{size: di.size, created: di.created}
	}

	return d
}

// addImage adds an image to the images map with deterministic metadata.
func (d *MockData) addImage(ref string) {
	if _, exists := d.images[ref]; exists {
		return
	}
	d.images[ref] = imageMeta{
		size:    deterministicSize(ref),
		created: deterministicCreated(ref),
	}
}

// GetServiceState returns the mock state for a service.
// Returns the stack status for running/paused stacks, "exited" for stopped stacks.
func (d *MockData) GetServiceState(stackName, svc, stackStatus string) string {
	switch stackStatus {
	case "running", "paused":
		key := stackName + "/" + svc
		if state, ok := d.serviceStates[key]; ok {
			return state
		}
		return stackStatus
	default:
		return "exited"
	}
}

// GetServiceHealth returns the mock health for a service.
func (d *MockData) GetServiceHealth(stackName, svc string) string {
	key := stackName + "/" + svc
	if health, ok := d.serviceHealth[key]; ok {
		return health
	}
	return ""
}

// GetRunningImage returns the image a service container is "actually running".
// If there's a mock.yaml override, returns that; otherwise returns the compose image.
func (d *MockData) GetRunningImage(stackName, svc string) string {
	key := stackName + "/" + svc
	if img, ok := d.runningImages[key]; ok {
		return img
	}
	if img, ok := d.serviceImages[key]; ok {
		return img
	}
	return "mock-image:latest"
}

// GetComposeImage returns the image declared in compose.yaml for a service.
func (d *MockData) GetComposeImage(stackName, svc string) string {
	key := stackName + "/" + svc
	if img, ok := d.serviceImages[key]; ok {
		return img
	}
	return "mock-image:latest"
}

// HasUpdateAvailable returns true if any service using this image has update_available set.
func (d *MockData) HasUpdateAvailable(imageRef string) bool {
	for key, hasUpdate := range d.updateFlags {
		if !hasUpdate {
			continue
		}
		// Check if this service uses this image (either compose or running)
		if img, ok := d.runningImages[key]; ok && img == imageRef {
			return true
		}
		if img, ok := d.serviceImages[key]; ok && img == imageRef {
			return true
		}
	}
	return false
}

// UpdateFlags returns the mock update flags map ("stackName/svc" → has update).
func (d *MockData) UpdateFlags() map[string]bool {
	return d.updateFlags
}

// SortedImages returns all image refs (excluding danglings) sorted.
func (d *MockData) SortedImages() []string {
	var refs []string
	for ref := range d.images {
		if !strings.HasPrefix(ref, "<dangling:") {
			refs = append(refs, ref)
		}
	}
	sort.Strings(refs)
	return refs
}

// SortedNetworks returns all network names sorted.
func (d *MockData) SortedNetworks() []string {
	var names []string
	for name := range d.networks {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// NetworkID returns the mock.yaml-driven ID for a network, or falls back
// to a deterministic 64-char hex hash (matching real Docker network IDs).
func (d *MockData) NetworkID(name string) string {
	if meta, ok := d.networks[name]; ok && meta.id != "" {
		return meta.id
	}
	h := mockHash(name)
	return h + h // 32 + 32 = 64 hex chars
}

// SortedVolumes returns all volume names sorted.
func (d *MockData) SortedVolumes() []string {
	var names []string
	for name := range d.volumes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// parseContainerKey extracts stack and service name from a container ID
// like "mock-stackName-serviceName-1". Handles hyphenated stack names by
// trying known stack/service combinations.
func (d *MockData) parseContainerKey(containerID string) (stack, service string, ok bool) {
	id := strings.TrimPrefix(containerID, "mock-")
	id = strings.TrimPrefix(id, "standalone-")

	// Try to match against known service keys
	for key := range d.serviceImages {
		parts := strings.SplitN(key, "/", 2)
		if len(parts) != 2 {
			continue
		}
		prefix := parts[0] + "-" + parts[1] + "-1"
		if id == prefix {
			return parts[0], parts[1], true
		}
	}
	return "", "", false
}

// --- Parsers ---

// parseComposeForMock extracts service/image/network/volume data from a compose file
// using simple line scanning (no YAML library needed).
func parseComposeForMock(path string) composeData {
	f, err := os.Open(path)
	if err != nil {
		return composeData{}
	}
	defer f.Close()

	var cd composeData
	var currentService *composeService
	topLevelVolumes := make(map[string]bool)

	section := "" // "services", "networks", "volumes", or ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimRight(line, " \t")

		if trimmed == "" || strings.HasPrefix(strings.TrimSpace(trimmed), "#") {
			continue
		}

		// Detect top-level sections
		if len(trimmed) > 0 && trimmed[0] != ' ' && trimmed[0] != '\t' {
			stripped := strings.TrimSuffix(strings.TrimSpace(trimmed), ":")
			switch stripped {
			case "services":
				section = "services"
			case "networks":
				section = "networks"
			case "volumes":
				section = "volumes"
			default:
				section = ""
			}
			currentService = nil
			continue
		}

		indent := countIndent(line)

		switch section {
		case "services":
			if indent == 2 && strings.HasSuffix(trimmed, ":") {
				// New service declaration
				name := strings.TrimSpace(strings.TrimSuffix(trimmed, ":"))
				cd.services = append(cd.services, composeService{name: name})
				currentService = &cd.services[len(cd.services)-1]
			} else if currentService != nil && indent >= 4 {
				fieldLine := strings.TrimSpace(trimmed)
				if strings.HasPrefix(fieldLine, "image:") {
					img := strings.TrimSpace(strings.TrimPrefix(fieldLine, "image:"))
					img = strings.Trim(img, "\"'")
					currentService.image = img
				} else if strings.HasPrefix(fieldLine, "- ") && indent == 6 {
					// Could be network or volume list item
					val := strings.TrimPrefix(fieldLine, "- ")
					val = strings.Trim(val, "\"'")
					// Determine parent key by scanning upward context
					// We track this via state
				}
			}

			// Parse per-service networks (list form: networks:\n  - name)
			if currentService != nil && indent == 4 && strings.TrimSpace(trimmed) == "networks:" {
				// Next lines at indent 6 starting with "- " are network refs
				// We handle this by setting a sub-section flag — but to keep
				// things simple, we'll use a different approach: re-scan
			}

			// Parse per-service volumes (list form: volumes:\n  - source:dest)
			// We handle these in a second pass approach below

		case "networks":
			if indent == 2 && strings.HasSuffix(trimmed, ":") {
				name := strings.TrimSpace(strings.TrimSuffix(trimmed, ":"))
				cd.networks = append(cd.networks, name)
			}

		case "volumes":
			if indent == 2 && strings.HasSuffix(trimmed, ":") {
				name := strings.TrimSpace(strings.TrimSuffix(trimmed, ":"))
				cd.volumes = append(cd.volumes, name)
				topLevelVolumes[name] = true
			}
		}
	}

	// Second pass: extract per-service networks and volumes
	f2, err := os.Open(path)
	if err != nil {
		return cd
	}
	defer f2.Close()

	svcIdx := -1
	inServiceNetworks := false
	inServiceVolumes := false
	inServicePorts := false
	section = ""
	scanner2 := bufio.NewScanner(f2)
	for scanner2.Scan() {
		line := scanner2.Text()
		trimmed := strings.TrimRight(line, " \t")
		if trimmed == "" || strings.HasPrefix(strings.TrimSpace(trimmed), "#") {
			continue
		}

		indent := countIndent(line)

		// Track top-level section
		if indent == 0 && len(trimmed) > 0 && trimmed[0] != ' ' && trimmed[0] != '\t' {
			stripped := strings.TrimSuffix(strings.TrimSpace(trimmed), ":")
			if stripped == "services" {
				section = "services"
			} else {
				section = ""
			}
			svcIdx = -1
			inServiceNetworks = false
			inServiceVolumes = false
			continue
		}

		if section != "services" {
			continue
		}

		if indent == 2 && strings.HasSuffix(trimmed, ":") {
			svcIdx++
			inServiceNetworks = false
			inServiceVolumes = false
			inServicePorts = false
			continue
		}

		if svcIdx < 0 || svcIdx >= len(cd.services) {
			continue
		}

		if indent == 4 {
			field := strings.TrimSpace(trimmed)
			if field == "networks:" {
				inServiceNetworks = true
				inServiceVolumes = false
				inServicePorts = false
				continue
			} else if field == "volumes:" {
				inServiceVolumes = true
				inServiceNetworks = false
				inServicePorts = false
				continue
			} else if field == "ports:" {
				inServicePorts = true
				inServiceNetworks = false
				inServiceVolumes = false
				continue
			} else if !strings.HasPrefix(field, "- ") {
				inServiceNetworks = false
				inServiceVolumes = false
				inServicePorts = false
			}
		}

		if indent == 6 {
			item := strings.TrimSpace(trimmed)
			if !strings.HasPrefix(item, "- ") {
				continue
			}
			val := strings.TrimPrefix(item, "- ")
			val = strings.Trim(val, "\"'")

			if inServiceNetworks {
				cd.services[svcIdx].networks = append(cd.services[svcIdx].networks, val)
			} else if inServiceVolumes {
				vr := parseVolumeRef(val, topLevelVolumes)
				cd.services[svcIdx].volumes = append(cd.services[svcIdx].volumes, vr)
			} else if inServicePorts {
				cd.services[svcIdx].ports = append(cd.services[svcIdx].ports, val)
			}
		}
	}

	return cd
}

// parseVolumeRef parses a volume reference like "pgdata:/var/lib/postgresql/data" or "./data:/app/data:ro".
func parseVolumeRef(ref string, topLevelVolumes map[string]bool) composeVolumeRef {
	parts := strings.SplitN(ref, ":", 3)
	vr := composeVolumeRef{}

	switch len(parts) {
	case 1:
		// Anonymous volume: just a path
		vr.destination = parts[0]
	case 2:
		vr.name = parts[0]
		vr.destination = parts[1]
	case 3:
		vr.name = parts[0]
		vr.destination = parts[1]
		vr.readOnly = parts[2] == "ro"
	}

	// Determine if named volume vs bind
	if vr.name != "" && !strings.HasPrefix(vr.name, ".") && !strings.HasPrefix(vr.name, "/") && !strings.HasPrefix(vr.name, "~") {
		if topLevelVolumes[vr.name] {
			vr.isNamed = true
		}
	}

	return vr
}

// parseMockYAML reads a mock.yaml sidecar file and returns overrides.
// Returns zero value if file doesn't exist.
func parseMockYAML(path string) mockOverrides {
	data, err := os.ReadFile(path)
	if err != nil {
		return mockOverrides{}
	}

	var raw mockYAMLFile
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return mockOverrides{}
	}

	mo := mockOverrides{
		status:   raw.Status,
		services: make(map[string]serviceOverrides, len(raw.Services)),
		networks: make(map[string]string, len(raw.Networks)),
	}

	for netName, nc := range raw.Networks {
		if nc.ID != "" {
			mo.networks[netName] = nc.ID
		}
	}

	for name, svc := range raw.Services {
		so := serviceOverrides{
			state:           svc.State,
			health:          svc.Health,
			runningImage:    svc.RunningImage,
			updateAvailable: svc.UpdateAvailable,
			command:         svc.Command,
			args:            svc.Args,
			env:             svc.Env,
			restartPolicy:   svc.RestartPolicy,
			logs:            svc.Logs,
		}
		if len(svc.Networks) > 0 {
			so.networks = make(map[string]endpointConfig, len(svc.Networks))
			for netName, ep := range svc.Networks {
				so.networks[netName] = endpointConfig{ip: ep.IP, mac: ep.MAC}
			}
		}
		mo.services[name] = so
	}

	return mo
}

// parseGlobalMockYAML reads the root-level mock.yaml in the stacks directory.
// This defines Docker resources that exist independently of any compose project
// (standalone networks, standalone containers, external stacks, dangling images).
// Returns zero value if file doesn't exist.
func parseGlobalMockYAML(path string) globalMockConfig {
	data, err := os.ReadFile(path)
	if err != nil {
		return globalMockConfig{}
	}

	var raw globalMockYAMLFile
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return globalMockConfig{}
	}

	cfg := globalMockConfig{
		networks: make(map[string]networkMeta, len(raw.Networks)),
	}

	for name, nd := range raw.Networks {
		driver := nd.Driver
		if driver == "" {
			driver = "bridge"
		}
		cfg.networks[name] = networkMeta{
			driver:   driver,
			scope:    "local",
			internal: nd.Internal,
			subnet:   nd.Subnet,
			gateway:  nd.Gateway,
			id:       nd.ID,
		}
	}

	for _, sc := range raw.StandaloneContainers {
		cfg.standalones = append(cfg.standalones, standaloneContainer{
			name:          sc.Name,
			image:         sc.Image,
			state:         sc.State,
			command:       sc.Command,
			restartPolicy: sc.RestartPolicy,
			network:       sc.Network,
			ip:            sc.IP,
			mac:           sc.MAC,
		})
	}

	if len(raw.ExternalStacks) > 0 {
		cfg.externals = make(map[string]map[string]externalServiceConfig, len(raw.ExternalStacks))
		for stackName, es := range raw.ExternalStacks {
			svcs := make(map[string]externalServiceConfig, len(es.Services))
			for svcName, svc := range es.Services {
				svcs[svcName] = externalServiceConfig{
					image:   svc.Image,
					command: svc.Command,
					ports:   svc.Ports,
					network: svc.Network,
					ip:      svc.IP,
					mac:     svc.MAC,
				}
			}
			cfg.externals[stackName] = svcs
		}
	}

	for _, di := range raw.DanglingImages {
		cfg.danglings = append(cfg.danglings, danglingImageConfig{
			id:      di.ID,
			size:    di.Size,
			created: di.Created,
		})
	}

	return cfg
}

// findComposeFilePath finds a compose file in a stack directory.
func findComposeFilePath(stackDir string) string {
	for _, name := range []string{"compose.yaml", "docker-compose.yaml", "docker-compose.yml", "compose.yml"} {
		path := filepath.Join(stackDir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// countIndent returns the number of leading spaces in a line.
func countIndent(line string) int {
	for i, c := range line {
		if c != ' ' && c != '\t' {
			return i
		}
	}
	return len(line)
}

// deterministicSize generates a realistic image size string from the image ref.
func deterministicSize(ref string) string {
	h := simpleHash(ref)
	// Generate sizes between 5 MiB and 2 GiB
	sizeMiB := 5 + (h % 2000)
	if sizeMiB >= 1024 {
		return fmt.Sprintf("%.1fGiB", float64(sizeMiB)/1024)
	}
	return fmt.Sprintf("%.1fMiB", float64(sizeMiB)+float64(h%10)/10)
}

// deterministicCreated generates a deterministic created timestamp from the image ref.
func deterministicCreated(ref string) string {
	h := simpleHash(ref)
	// Generate dates in 2025-2026 range
	month := 1 + (h % 12)
	day := 1 + (h/12)%28
	hour := (h / 336) % 24
	year := 2025 + (h/8064)%2
	return fmt.Sprintf("%d-%02d-%02dT%02d:00:00Z", year, month, day, hour)
}

// simpleHash returns a simple non-cryptographic hash of a string.
func simpleHash(s string) uint64 {
	var h uint64 = 5381
	for _, c := range s {
		h = h*33 + uint64(c)
	}
	return h
}

// knownWorkingDirs maps well-known image base names to their typical WORKDIR.
var knownWorkingDirs = map[string]string{
	"nginx":           "/usr/share/nginx/html",
	"httpd":           "/usr/local/apache2/htdocs",
	"redis":           "/data",
	"postgres":        "/",
	"mysql":           "/",
	"mariadb":         "/",
	"mongo":           "/data/db",
	"wordpress":       "/var/www/html",
	"grafana":         "/usr/share/grafana",
	"node":            "/",
	"python":          "/",
	"alpine":          "/",
	"busybox":         "/",
	"ubuntu":          "/",
	"debian":          "/",
	"golang":          "/go",
	"ruby":            "/usr/local/bundle",
	"php":             "/var/www/html",
	"elasticsearch":   "/usr/share/elasticsearch",
	"rabbitmq":        "/",
	"memcached":       "/",
	"traefik":         "/",
	"caddy":           "/srv",
	"vault":           "/",
	"consul":          "/",
	"minio":           "/data",
	"portainer-ce":    "/",
	"watchtower":      "/",
	"home-assistant":  "/config",
	"portainer":       "/",
}

// workingDirForImage returns the WORKDIR for a known image, or "" for unknown.
func workingDirForImage(imageRef string) string {
	// Extract base name: "grafana/grafana:latest" → "grafana"
	name := imageRef
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	if idx := strings.Index(name, ":"); idx >= 0 {
		name = name[:idx]
	}
	if dir, ok := knownWorkingDirs[name]; ok {
		return dir
	}
	return ""
}

// --- Log Template Parsing ---

// defaultBaseTime is the hardcoded fallback when no base_time is specified.
var defaultBaseTime = time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

// hasContent returns true if the YAML has any log content defined.
func (s serviceLogsYAML) hasContent() bool {
	return len(s.Startup) > 0 || len(s.Heartbeat.Lines) > 0 || len(s.Shutdown) > 0 || s.BaseTime != ""
}

// resolve converts a serviceLogsYAML into a ready-to-use ServiceLogs.
func (s serviceLogsYAML) resolve() *ServiceLogs {
	sl := &ServiceLogs{
		BaseTime:  defaultBaseTime,
		Startup:   s.Startup,
		Heartbeat: s.Heartbeat.Lines,
		Shutdown:  s.Shutdown,
		Interval:  3 * time.Second,
	}
	if s.BaseTime != "" {
		if t, err := time.Parse(time.RFC3339Nano, s.BaseTime); err == nil {
			sl.BaseTime = t
		} else if t, err := time.Parse("2006-01-02T15:04:05.000Z", s.BaseTime); err == nil {
			sl.BaseTime = t
		}
	}
	if s.Heartbeat.Interval != "" {
		if d, err := time.ParseDuration(s.Heartbeat.Interval); err == nil {
			sl.Interval = d
		}
	}
	return sl
}

// parseLogTemplates parses log-templates.yaml into a map of image base → ServiceLogs.
// The file uses a two-level structure: a top-level "base_time" string, and all other
// top-level keys are image names mapping to serviceLogsYAML.
func parseLogTemplates(path string) map[string]*ServiceLogs {
	templates := make(map[string]*ServiceLogs)

	data, err := os.ReadFile(path)
	if err != nil {
		return templates
	}

	// First pass: unmarshal into map[string]yaml.Node to get all keys
	var raw map[string]yaml.Node
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return templates
	}

	// Extract global base_time
	globalBaseTime := defaultBaseTime
	if node, ok := raw["base_time"]; ok {
		var btStr string
		if err := node.Decode(&btStr); err == nil {
			if t, err := time.Parse(time.RFC3339Nano, btStr); err == nil {
				globalBaseTime = t
			} else if t, err := time.Parse("2006-01-02T15:04:05.000Z", btStr); err == nil {
				globalBaseTime = t
			}
		}
	}

	// Second pass: decode each image template
	for key, node := range raw {
		if key == "base_time" {
			continue
		}
		var sly serviceLogsYAML
		if err := node.Decode(&sly); err != nil {
			continue
		}
		sl := sly.resolve()
		// Apply per-template base_time override, falling back to global
		if sly.BaseTime == "" {
			sl.BaseTime = globalBaseTime
		}
		templates[key] = sl
	}

	return templates
}

// extractImageBaseName returns the base name of a Docker image ref.
// "grafana/grafana:latest" → "grafana", "nginx:1.25" → "nginx"
func extractImageBaseName(imageRef string) string {
	name := imageRef
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	if idx := strings.Index(name, ":"); idx >= 0 {
		name = name[:idx]
	}
	return name
}

// GetServiceLogs returns the resolved log definition for a service.
// Resolution order: per-service override → image template → default template.
func (d *MockData) GetServiceLogs(stackName, svc string) *ServiceLogs {
	key := stackName + "/" + svc
	if sl, ok := d.serviceLogs[key]; ok {
		return sl
	}
	// Fall back to image template
	if img, ok := d.serviceImages[key]; ok {
		imageBase := extractImageBaseName(img)
		if tmpl, ok := d.logTemplates[imageBase]; ok {
			return tmpl
		}
	}
	// Fall back to default template
	if tmpl, ok := d.logTemplates["default"]; ok {
		return tmpl
	}
	// Absolute fallback
	return &ServiceLogs{
		BaseTime: defaultBaseTime,
		Startup:  []string{"Service starting", "Service ready"},
		Interval: 3 * time.Second,
	}
}

// ExpandLogTemplate expands template variables in a log line.
// Variables: {{.Timestamp}}, {{.N}}, {{.Image}}
func ExpandLogTemplate(s string, n int, baseTime time.Time, interval time.Duration, imageBase string) string {
	if !strings.ContainsRune(s, '{') {
		return s // fast path
	}
	ts := baseTime.Add(time.Duration(n) * interval).Format("2006-01-02T15:04:05.000Z")
	s = strings.ReplaceAll(s, "{{.Timestamp}}", ts)
	s = strings.ReplaceAll(s, "{{.N}}", strconv.Itoa(n))
	s = strings.ReplaceAll(s, "{{.Image}}", imageBase)
	return s
}
