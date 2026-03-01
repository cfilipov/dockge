package mock

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cfilipov/dockge/internal/docker"
)

// FakeDaemon is an HTTP server on a Unix socket that implements the Docker
// Engine API using in-memory MockState + MockData. This allows the real
// SDKClient to connect to it exactly as it would to a real Docker daemon.
type FakeDaemon struct {
	state        *MockState
	data         *MockData
	world        *MockWorld
	stacksDir    string
	stacksSource string // pristine source dir for reset file restoration (empty = no file reset)
	listener     net.Listener
	server       *http.Server

	// Events infrastructure: subscribers receive state-change notifications.
	eventsMu    sync.Mutex
	eventSubs   map[int]chan eventMessage
	nextSubID   int
}

// eventMessage is a Docker-style event for JSON streaming.
type eventMessage struct {
	Status string            `json:"status"`
	ID     string            `json:"id"`
	Type   string            `json:"Type"`
	Action string            `json:"Action"`
	Actor  eventActor        `json:"Actor"`
	Time   int64             `json:"time"`
	TimeNano int64           `json:"timeNano"`
}

type eventActor struct {
	ID         string            `json:"ID"`
	Attributes map[string]string `json:"Attributes"`
}

// StartFakeDaemon creates and starts a fake Docker daemon on a Unix socket.
// stacksSource is the pristine source directory for file restoration on reset
// (pass "" to skip file restoration, e.g. in unit tests).
// Returns the socket path for DOCKER_HOST, a cleanup function, and any error.
func StartFakeDaemon(state *MockState, data *MockData, stacksDir, stacksSource string) (socketPath string, cleanup func(), err error) {
	// Create temp directory for the socket
	tmpDir, err := os.MkdirTemp("", "dockge-mock-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temp dir: %w", err)
	}

	sockPath := filepath.Join(tmpDir, "docker.sock")
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, fmt.Errorf("listen unix: %w", err)
	}

	world := BuildMockWorld(data, state, stacksDir)

	fd := &FakeDaemon{
		state:        state,
		data:         data,
		world:        world,
		stacksDir:    stacksDir,
		stacksSource: stacksSource,
		listener:     listener,
		eventSubs:    make(map[int]chan eventMessage),
	}

	mux := http.NewServeMux()
	fd.registerRoutes(mux)

	fd.server = &http.Server{Handler: fd.stripVersionPrefix(mux)}

	go func() {
		if err := fd.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			slog.Error("fake daemon serve", "err", err)
		}
	}()

	cleanupFn := func() {
		fd.server.Close()
		listener.Close()
		os.RemoveAll(tmpDir)
	}

	return sockPath, cleanupFn, nil
}

// StartFakeDaemonOnSocket creates and starts a fake Docker daemon on a
// caller-specified Unix socket path. Used by cmd/mock-daemon for external
// daemon operation. Returns a cleanup function and any error.
func StartFakeDaemonOnSocket(state *MockState, data *MockData, stacksDir, stacksSource, sockPath string) (cleanup func(), err error) {
	// Ensure parent directory exists
	if dir := filepath.Dir(sockPath); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create socket dir: %w", err)
		}
	}

	// Remove stale socket file if it exists
	os.Remove(sockPath)

	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, fmt.Errorf("listen unix: %w", err)
	}

	world := BuildMockWorld(data, state, stacksDir)

	fd := &FakeDaemon{
		state:        state,
		data:         data,
		world:        world,
		stacksDir:    stacksDir,
		stacksSource: stacksSource,
		listener:     listener,
		eventSubs:    make(map[int]chan eventMessage),
	}

	mux := http.NewServeMux()
	fd.registerRoutes(mux)

	fd.server = &http.Server{Handler: fd.stripVersionPrefix(mux)}

	go func() {
		if err := fd.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			slog.Error("fake daemon serve", "err", err)
		}
	}()

	cleanupFn := func() {
		fd.server.Close()
		listener.Close()
		os.Remove(sockPath)
	}

	return cleanupFn, nil
}

// stripVersionPrefix returns middleware that strips /v{version}/ prefix from requests.
// Docker SDK sends requests like /v1.47/containers/json.
func (fd *FakeDaemon) stripVersionPrefix(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if len(path) > 2 && path[0] == '/' && path[1] == 'v' {
			// Strip /v1.47/ prefix
			if idx := strings.IndexByte(path[2:], '/'); idx >= 0 {
				r.URL.Path = path[2+idx:]
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (fd *FakeDaemon) registerRoutes(mux *http.ServeMux) {
	// Ping
	mux.HandleFunc("HEAD /_ping", fd.handlePing)
	mux.HandleFunc("GET /_ping", fd.handlePing)

	// Containers
	mux.HandleFunc("GET /containers/json", fd.handleContainerList)
	mux.HandleFunc("GET /containers/{id}/json", fd.handleContainerInspect)
	mux.HandleFunc("GET /containers/{id}/stats", fd.handleContainerStats)
	mux.HandleFunc("GET /containers/{id}/top", fd.handleContainerTop)
	mux.HandleFunc("GET /containers/{id}/logs", fd.handleContainerLogs)

	// Images — name can contain slashes (e.g., "library/nginx"), so we can't
	// use {name...} in the middle of a pattern. Use prefix-based routing instead.
	mux.HandleFunc("GET /images/json", fd.handleImageList)
	mux.HandleFunc("POST /images/prune", fd.handleImagePrune)
	mux.HandleFunc("GET /images/", fd.handleImageRoute)

	// Distribution — same slash issue as images.
	mux.HandleFunc("GET /distribution/", fd.handleDistributionRoute)

	// Networks
	mux.HandleFunc("GET /networks", fd.handleNetworkList)
	mux.HandleFunc("GET /networks/{id}", fd.handleNetworkInspect)

	// Volumes
	mux.HandleFunc("GET /volumes", fd.handleVolumeList)
	mux.HandleFunc("GET /volumes/{name}", fd.handleVolumeInspect)

	// Events
	mux.HandleFunc("GET /events", fd.handleEvents)

	// Custom mock state endpoints
	mux.HandleFunc("POST /_mock/state/{stack}/{service}", fd.handleMockServiceStateSet)
	mux.HandleFunc("POST /_mock/state/{stack}", fd.handleMockStateSet)
	mux.HandleFunc("DELETE /_mock/state/{stack}", fd.handleMockStateDelete)
	mux.HandleFunc("POST /_mock/reset", fd.handleMockReset)
	mux.HandleFunc("GET /_mock/logs/{stack}/{service}", fd.handleMockLogs)
}

// handleImageRoute routes GET /images/{name}/json and GET /images/{name}/history
// where name may contain slashes (e.g., "library/nginx:latest").
func (fd *FakeDaemon) handleImageRoute(w http.ResponseWriter, r *http.Request) {
	// Path is /images/{name}/json or /images/{name}/history
	path := strings.TrimPrefix(r.URL.Path, "/images/")
	if strings.HasSuffix(path, "/json") {
		name := strings.TrimSuffix(path, "/json")
		r.SetPathValue("name", name)
		fd.handleImageInspect(w, r)
	} else if strings.HasSuffix(path, "/history") {
		name := strings.TrimSuffix(path, "/history")
		r.SetPathValue("name", name)
		fd.handleImageHistory(w, r)
	} else {
		http.NotFound(w, r)
	}
}

// handleDistributionRoute routes GET /distribution/{name}/json
// where name may contain slashes.
func (fd *FakeDaemon) handleDistributionRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/distribution/")
	if strings.HasSuffix(path, "/json") {
		name := strings.TrimSuffix(path, "/json")
		r.SetPathValue("name", name)
		fd.handleDistributionInspect(w, r)
	} else {
		http.NotFound(w, r)
	}
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// --- Ping ---

func (fd *FakeDaemon) handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Api-Version", "1.47")
	w.Header().Set("Docker-Experimental", "false")
	w.Header().Set("Ostype", "linux")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// --- Containers ---

// containerJSON matches the Docker SDK container.Summary type fields.
type containerJSON struct {
	ID         string                       `json:"Id"`
	Names      []string                     `json:"Names"`
	Image      string                       `json:"Image"`
	ImageID    string                       `json:"ImageID"`
	Command    string                       `json:"Command"`
	Created    int64                        `json:"Created"`
	State      string                       `json:"State"`
	Status     string                       `json:"Status"`
	Labels     map[string]string            `json:"Labels"`
	Mounts     []mountJSON                  `json:"Mounts"`
	NetworkSettings *networkSettingsJSON    `json:"NetworkSettings"`
}

type mountJSON struct {
	Type        string `json:"Type"`
	Name        string `json:"Name,omitempty"`
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
	Mode        string `json:"Mode"`
	RW          bool   `json:"RW"`
}

type networkSettingsJSON struct {
	Networks map[string]endpointJSON `json:"Networks"`
}

type endpointJSON struct {
	IPAddress   string `json:"IPAddress"`
	IPPrefixLen int    `json:"IPPrefixLen"`
	Gateway     string `json:"Gateway"`
	MacAddress  string `json:"MacAddress"`
	NetworkID   string `json:"NetworkID"`
}

func (fd *FakeDaemon) handleContainerList(w http.ResponseWriter, r *http.Request) {
	allParam := r.URL.Query().Get("all")
	all := allParam == "1" || allParam == "true"

	// Parse filters — Docker SDK sends either:
	//   {"label":["key=val"]}  (array form) or
	//   {"label":{"key=val":true}}  (map form)
	projectFilter := ""
	filtersParam := r.URL.Query().Get("filters")
	if filtersParam != "" {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal([]byte(filtersParam), &raw); err == nil {
			if labelRaw, ok := raw["label"]; ok {
				projectFilter = extractProjectFilter(labelRaw)
			}
		}
	}

	containers := fd.world.ContainerList(all, projectFilter)
	writeJSON(w, http.StatusOK, containers)
}

func buildStatusString(state, health string) string {
	switch state {
	case "running":
		base := "Up 2 hours"
		if health == "unhealthy" {
			base += " (unhealthy)"
		} else if health == "healthy" {
			base += " (healthy)"
		} else if health == "starting" {
			base += " (health: starting)"
		}
		return base
	case "paused":
		return "Up 2 hours (Paused)"
	case "exited":
		return "Exited (0) 2 hours ago"
	default:
		return "Created"
	}
}

// buildMounts, buildEndpoints, buildPortBindings removed — now in mockworld.go

// containerInspectJSON matches the Docker SDK container.InspectResponse fields.
type containerInspectJSON struct {
	ID           string             `json:"Id"`
	Created      string             `json:"Created"`
	Name         string             `json:"Name"`
	Path         string             `json:"Path"`
	Args         []string           `json:"Args"`
	State        *containerStateJSON `json:"State"`
	RestartCount int                `json:"RestartCount"`
	Image        string             `json:"Image"`
	Config       *containerConfigJSON `json:"Config"`
	HostConfig   *hostConfigJSON    `json:"HostConfig"`
	Mounts       []mountJSON        `json:"Mounts"`
	NetworkSettings *inspectNetworkSettingsJSON `json:"NetworkSettings"`
}

type containerStateJSON struct {
	Status     string `json:"Status"`
	Running    bool   `json:"Running"`
	Paused     bool   `json:"Paused"`
	Restarting bool   `json:"Restarting"`
	OOMKilled  bool   `json:"OOMKilled"`
	Dead       bool   `json:"Dead"`
	Pid        int    `json:"Pid"`
	ExitCode   int    `json:"ExitCode"`
	StartedAt  string `json:"StartedAt"`
	FinishedAt string `json:"FinishedAt"`
}

type containerConfigJSON struct {
	Hostname   string   `json:"Hostname"`
	Image      string   `json:"Image"`
	Cmd        []string `json:"Cmd"`
	WorkingDir string   `json:"WorkingDir"`
	User       string   `json:"User"`
	Env        []string `json:"Env"`
	Tty        bool     `json:"Tty"`
}

type hostConfigJSON struct {
	RestartPolicy restartPolicyJSON `json:"RestartPolicy"`
}

type restartPolicyJSON struct {
	Name             string `json:"Name"`
	MaximumRetryCount int   `json:"MaximumRetryCount"`
}

type portBindingJSON struct {
	HostIp   string `json:"HostIp"`
	HostPort string `json:"HostPort"`
}

type inspectNetworkSettingsJSON struct {
	Ports    map[string][]portBindingJSON    `json:"Ports"`
	Networks map[string]inspectEndpointJSON `json:"Networks"`
}

type inspectEndpointJSON struct {
	IPAddress          string `json:"IPAddress"`
	IPPrefixLen        int    `json:"IPPrefixLen"`
	IPv6Gateway        string `json:"IPv6Gateway"`
	GlobalIPv6Address  string `json:"GlobalIPv6Address"`
	GlobalIPv6PrefixLen int   `json:"GlobalIPv6PrefixLen"`
	Gateway            string `json:"Gateway"`
	MacAddress         string `json:"MacAddress"`
	Aliases            []string `json:"Aliases"`
	NetworkID          string `json:"NetworkID"`
}

func (fd *FakeDaemon) handleContainerInspect(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	resp, ok := fd.world.ContainerInspect(id)
	if !ok {
		http.Error(w, fmt.Sprintf(`{"message":"No such container: %s"}`, id), http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// --- Container Stats ---

// statsJSON matches the Docker SDK container.StatsResponse fields used by SDKClient.
type statsJSON struct {
	Read     string       `json:"read"`
	PreRead  string       `json:"preread"`
	CPUStats cpuStatsJSON `json:"cpu_stats"`
	PreCPUStats cpuStatsJSON `json:"precpu_stats"`
	MemoryStats memStatsJSON `json:"memory_stats"`
	Networks map[string]netStatsJSON `json:"networks"`
	BlkioStats blkioStatsJSON `json:"blkio_stats"`
	PidsStats  pidsStatsJSON  `json:"pids_stats"`
}

type cpuStatsJSON struct {
	CPUUsage    cpuUsageJSON `json:"cpu_usage"`
	SystemUsage uint64       `json:"system_cpu_usage"`
	OnlineCPUs  uint32       `json:"online_cpus"`
}

type cpuUsageJSON struct {
	TotalUsage uint64 `json:"total_usage"`
}

type memStatsJSON struct {
	Usage uint64            `json:"usage"`
	Limit uint64            `json:"limit"`
	Stats map[string]uint64 `json:"stats"`
}

type netStatsJSON struct {
	RxBytes uint64 `json:"rx_bytes"`
	TxBytes uint64 `json:"tx_bytes"`
}

type blkioStatsJSON struct {
	IoServiceBytesRecursive []blkioEntryJSON `json:"io_service_bytes_recursive"`
}

type blkioEntryJSON struct {
	Op    string `json:"op"`
	Value uint64 `json:"value"`
}

type pidsStatsJSON struct {
	Current uint64 `json:"current"`
}

func (fd *FakeDaemon) handleContainerStats(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Determine container state from MockWorld (dynamically resolved)
	svcState := fd.world.GetContainerState(id)

	// Docker SDK expects a JSON body, streamed (stream=false returns one shot).
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	now := time.Now()
	stats := statsJSON{
		Read:    now.Format(time.RFC3339Nano),
		PreRead: now.Add(-time.Second).Format(time.RFC3339Nano),
	}

	switch svcState {
	case "exited", "inactive":
		// Exited containers: all zeros (matches real Docker behavior)
		stats.CPUStats = cpuStatsJSON{OnlineCPUs: 4}
		stats.PreCPUStats = cpuStatsJSON{OnlineCPUs: 4}
		stats.MemoryStats = memStatsJSON{
			Limit: 2147483648,
			Stats: map[string]uint64{"cache": 0},
		}

	case "paused":
		// Paused containers: 0 CPU, non-zero memory (process is frozen in RAM)
		stats.CPUStats = cpuStatsJSON{
			CPUUsage:    cpuUsageJSON{TotalUsage: 100000000},
			SystemUsage: 83400000000,
			OnlineCPUs:  4,
		}
		stats.PreCPUStats = cpuStatsJSON{
			CPUUsage:    cpuUsageJSON{TotalUsage: 100000000}, // same as current → 0% CPU
			SystemUsage: 83300000000,
			OnlineCPUs:  4,
		}
		h := simpleHash(id)
		memUsage := 10*1024*1024 + uint64(h%50)*1024*1024 // 10-60 MiB
		stats.MemoryStats = memStatsJSON{
			Usage: memUsage,
			Limit: 2147483648,
			Stats: map[string]uint64{"cache": 0},
		}
		stats.PidsStats = pidsStatsJSON{Current: 3 + h%10}

	default: // "running"
		// Vary stats per container using deterministic hash
		h := simpleHash(id)
		cpuDelta := 50000 + uint64(h%500000)       // 50K-550K ns delta → ~0.02%-0.22% CPU
		memUsage := 10*1024*1024 + (h%200)*1024*1024 // 10-210 MiB
		rxBytes := 1000 + h%100000
		txBytes := 500 + (h/100)%50000
		pids := 2 + h%20

		stats.CPUStats = cpuStatsJSON{
			CPUUsage:    cpuUsageJSON{TotalUsage: 100000000 + cpuDelta},
			SystemUsage: 83400000000,
			OnlineCPUs:  4,
		}
		stats.PreCPUStats = cpuStatsJSON{
			CPUUsage:    cpuUsageJSON{TotalUsage: 100000000},
			SystemUsage: 83300000000,
			OnlineCPUs:  4,
		}
		stats.MemoryStats = memStatsJSON{
			Usage: memUsage,
			Limit: 2147483648,
			Stats: map[string]uint64{"cache": 0},
		}
		stats.Networks = map[string]netStatsJSON{
			"eth0": {RxBytes: rxBytes, TxBytes: txBytes},
		}
		stats.BlkioStats = blkioStatsJSON{
			IoServiceBytesRecursive: []blkioEntryJSON{
				{Op: "read", Value: h % 10000000},
				{Op: "write", Value: (h / 10) % 5000000},
			},
		}
		stats.PidsStats = pidsStatsJSON{Current: pids}
	}

	json.NewEncoder(w).Encode(stats)
}

// --- Container Top ---

type topResponseJSON struct {
	Titles    []string   `json:"Titles"`
	Processes [][]string `json:"Processes"`
}

func (fd *FakeDaemon) handleContainerTop(w http.ResponseWriter, r *http.Request) {
	resp := topResponseJSON{
		Titles: []string{"PID", "USER", "COMMAND"},
		Processes: [][]string{
			{"1", "root", "nginx: master process nginx -g daemon off;"},
			{"29", "nginx", "nginx: worker process"},
			{"30", "nginx", "nginx: worker process"},
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- Container Logs ---

func (fd *FakeDaemon) handleContainerLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	follow := r.URL.Query().Get("follow") == "1" || r.URL.Query().Get("follow") == "true"

	c, ok := fd.world.GetContainer(id)
	if !ok {
		http.NotFound(w, r)
		return
	}

	// Resolve the canonical container ID for event matching.
	// The URL path may contain the container name (e.g. "stack-svc-1")
	// but events use the container ID (e.g. "mock-stack-svc-1").
	containerID := c.ID

	logs := fd.data.GetServiceLogs(c.StackName, c.ServiceName)
	imageBase := extractImageBase(c.Image)

	isRunning := fd.world.GetContainerState(containerID) == "running"

	// Real Docker returns 409 when following logs of a non-running container.
	if follow && !isRunning {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		fmt.Fprintf(w, `{"message":"container %s is not running"}`, containerID)
		return
	}

	// Docker logs use stdcopy multiplexing for non-TTY containers.
	w.Header().Set("Content-Type", "application/vnd.docker.raw-stream")
	w.WriteHeader(http.StatusOK)

	// Emit startup lines
	for i, line := range logs.Startup {
		expanded := ExpandLogTemplate(line, i, logs.BaseTime, logs.Interval, imageBase)
		writeStdcopyLine(w, expanded+"\n")
	}

	// If the container is already exited, emit shutdown logs too.
	if !isRunning {
		for i, line := range logs.Shutdown {
			expanded := ExpandLogTemplate(line, i, logs.BaseTime, logs.Interval, imageBase)
			writeStdcopyLine(w, expanded+"\n")
		}
		return
	}

	if !follow {
		return
	}
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Subscribe to events for shutdown detection
	subID, eventCh := fd.subscribeEvents()
	defer fd.unsubscribeEvents(subID)

	interval := logs.Interval
	if interval == 0 {
		interval = 3 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	n := 0
	for {
		select {
		case <-r.Context().Done():
			return
		case evt := <-eventCh:
			if evt.ID == containerID && evt.Action == "die" {
				for i, line := range logs.Shutdown {
					expanded := ExpandLogTemplate(line, i, logs.BaseTime, logs.Interval, imageBase)
					writeStdcopyLine(w, expanded+"\n")
				}
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
				return
			}
		case <-ticker.C:
			// Skip heartbeat if the container is no longer running —
			// the "die" event handler above will emit shutdown logs and return.
			if fd.world.GetContainerState(containerID) != "running" {
				continue
			}
			if len(logs.Heartbeat) == 0 {
				continue
			}
			line := logs.Heartbeat[n%len(logs.Heartbeat)]
			expanded := ExpandLogTemplate(line, n, logs.BaseTime, interval, imageBase)
			n++
			if err := writeStdcopyLine(w, expanded+"\n"); err != nil {
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

// extractImageBase returns the base name of a Docker image (e.g. "nginx" from "nginx:latest").
func extractImageBase(imageRef string) string {
	name := imageRef
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	if idx := strings.Index(name, ":"); idx >= 0 {
		name = name[:idx]
	}
	return name
}


// writeStdcopyLine writes a line with Docker stdcopy multiplexing header.
// Format: [stream_type(1 byte)][0 0 0][size(4 bytes big-endian)][payload]
func writeStdcopyLine(w io.Writer, line string) error {
	header := make([]byte, 8)
	header[0] = 1 // stdout
	binary.BigEndian.PutUint32(header[4:], uint32(len(line)))
	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err := w.Write([]byte(line))
	return err
}

// --- Images ---

type imageJSON struct {
	ID          string   `json:"Id"`
	ParentID    string   `json:"ParentId"`
	RepoTags    []string `json:"RepoTags"`
	RepoDigests []string `json:"RepoDigests"`
	Created     int64    `json:"Created"`
	Size        int64    `json:"Size"`
	SharedSize  int64    `json:"SharedSize"`
	Containers  int64    `json:"Containers"`
}

func (fd *FakeDaemon) handleImageList(w http.ResponseWriter, r *http.Request) {
	// Count containers per image
	containers := fd.world.ContainerList(true, "")
	countByImageID := make(map[string]int)
	for _, c := range containers {
		countByImageID[c.ImageID]++
	}

	refs := fd.data.SortedImages()
	result := make([]imageJSON, 0, len(refs)+2)

	for _, ref := range refs {
		meta := fd.data.images[ref]
		hash := mockHash(ref)
		id := fmt.Sprintf("sha256:%s%s", hash, hash)

		created, _ := time.Parse(time.RFC3339, meta.created)
		sizeBytes := parseSizeToBytes(meta.size)

		result = append(result, imageJSON{
			ID:       id,
			RepoTags: []string{ref},
			Created:  created.Unix(),
			Size:     sizeBytes,
			Containers: int64(countByImageID[id]),
		})
	}

	// Dangling images from MockData
	for i, di := range fd.data.danglingImages {
		hash := mockHash(fmt.Sprintf("dangling-%d-%s", i, di.id))
		created, _ := time.Parse(time.RFC3339, di.created)
		result = append(result, imageJSON{
			ID:       fmt.Sprintf("sha256:%s%s", hash, hash),
			RepoTags: []string{},
			Created:  created.Unix(),
			Size:     parseSizeToBytes(di.size),
		})
	}

	writeJSON(w, http.StatusOK, result)
}

// imageInspectJSON matches the Docker SDK image.InspectResponse fields.
type imageInspectJSON struct {
	ID           string   `json:"Id"`
	RepoTags     []string `json:"RepoTags"`
	RepoDigests  []string `json:"RepoDigests"`
	Created      string   `json:"Created"`
	Size         int64    `json:"Size"`
	Architecture string   `json:"Architecture"`
	Os           string   `json:"Os"`
	Config       *imageConfigJSON `json:"Config,omitempty"`
}

type imageConfigJSON struct {
	WorkingDir string `json:"WorkingDir"`
}

func (fd *FakeDaemon) handleImageInspect(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	hash := mockHash(name)
	id := fmt.Sprintf("sha256:%s%s", hash, hash)

	// Compute RepoDigests
	repo, tag := splitImageRef(name)
	digestHash := mockHash(repo + ":" + tag)
	digest := fmt.Sprintf("sha256:%s%s", digestHash, digestHash)
	repoDigest := fmt.Sprintf("%s@%s", repo, digest)

	meta, hasMeta := fd.data.images[name]
	created := "2026-02-18T00:00:00Z"
	var sizeBytes int64
	if hasMeta {
		created = meta.created
		sizeBytes = parseSizeToBytes(meta.size)
	}

	wd := workingDirForImage(name)

	resp := imageInspectJSON{
		ID:           id,
		RepoTags:     []string{name},
		RepoDigests:  []string{repoDigest},
		Created:      created,
		Size:         sizeBytes,
		Architecture: "amd64",
		Os:           "linux",
		Config:       &imageConfigJSON{WorkingDir: wd},
	}

	writeJSON(w, http.StatusOK, resp)
}

// imageHistoryJSON matches image.HistoryResponseItem.
type imageHistoryJSON struct {
	ID        string `json:"Id"`
	Created   int64  `json:"Created"`
	Size      int64  `json:"Size"`
	CreatedBy string `json:"CreatedBy"`
}

func (fd *FakeDaemon) handleImageHistory(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	hash := mockHash(name)
	id := fmt.Sprintf("sha256:%s%s", hash, hash)

	meta, hasMeta := fd.data.images[name]
	created := time.Date(2026, 2, 18, 0, 0, 0, 0, time.UTC)
	if hasMeta {
		if t, err := time.Parse(time.RFC3339, meta.created); err == nil {
			created = t
		}
	}

	layers := generateLayers(name, id[:19], created.Format(time.RFC3339))
	result := make([]imageHistoryJSON, 0, len(layers))

	for _, l := range layers {
		layerCreated := created
		if t, err := time.Parse(time.RFC3339, l.Created); err == nil {
			layerCreated = t
		}
		layerID := l.ID
		if layerID == "<missing>" {
			layerID = "<missing>"
		}
		result = append(result, imageHistoryJSON{
			ID:        layerID,
			Created:   layerCreated.Unix(),
			Size:      parseSizeToBytes(l.Size),
			CreatedBy: l.Command,
		})
	}

	writeJSON(w, http.StatusOK, result)
}

// imagePruneJSON matches image.PruneReport.
type imagePruneJSON struct {
	ImagesDeleted  []interface{} `json:"ImagesDeleted"`
	SpaceReclaimed uint64        `json:"SpaceReclaimed"`
}

func (fd *FakeDaemon) handleImagePrune(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, imagePruneJSON{
		ImagesDeleted:  []interface{}{},
		SpaceReclaimed: 0,
	})
}

// --- Distribution ---

// distributionJSON matches registry.DistributionInspect.
type distributionJSON struct {
	Descriptor descriptorJSON `json:"Descriptor"`
}

type descriptorJSON struct {
	MediaType string `json:"mediaType"`
	Digest    string `json:"digest"`
	Size      int64  `json:"size"`
}

func (fd *FakeDaemon) handleDistributionInspect(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	repo, tag := splitImageRef(name)

	var hash string
	if fd.data.HasUpdateAvailable(name) {
		hash = mockHash(repo + ":" + tag + ":remote-newer")
	} else {
		hash = mockHash(repo + ":" + tag)
	}
	digest := fmt.Sprintf("sha256:%s%s", hash, hash)

	writeJSON(w, http.StatusOK, distributionJSON{
		Descriptor: descriptorJSON{
			MediaType: "application/vnd.docker.distribution.manifest.v2+json",
			Digest:    digest,
			Size:      1234,
		},
	})
}

// --- Networks ---

type networkJSON struct {
	Name       string              `json:"Name"`
	ID         string              `json:"Id"`
	Created    string              `json:"Created"`
	Scope      string              `json:"Scope"`
	Driver     string              `json:"Driver"`
	EnableIPv6 bool                `json:"EnableIPv6"`
	Internal   bool                `json:"Internal"`
	Attachable bool                `json:"Attachable"`
	Ingress    bool                `json:"Ingress"`
	IPAM       networkIPAMJSON     `json:"IPAM"`
	Containers map[string]networkContainerJSON `json:"Containers"`
}

type networkIPAMJSON struct {
	Driver string          `json:"Driver"`
	Config []ipamConfigJSON `json:"Config"`
}

type ipamConfigJSON struct {
	Subnet  string `json:"Subnet"`
	Gateway string `json:"Gateway"`
}

type networkContainerJSON struct {
	Name        string `json:"Name"`
	EndpointID  string `json:"EndpointID"`
	MacAddress  string `json:"MacAddress"`
	IPv4Address string `json:"IPv4Address"`
	IPv6Address string `json:"IPv6Address"`
}

func (fd *FakeDaemon) handleNetworkList(w http.ResponseWriter, r *http.Request) {
	result := fd.world.NetworkList()
	writeJSON(w, http.StatusOK, result)
}

func (fd *FakeDaemon) handleNetworkInspect(w http.ResponseWriter, r *http.Request) {
	networkID := r.PathValue("id")

	resp, ok := fd.world.NetworkInspect(networkID)
	if !ok {
		http.Error(w, fmt.Sprintf(`{"message":"network %s not found"}`, networkID), http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// --- Volumes ---

type volumeListJSON struct {
	Volumes  []volumeJSON `json:"Volumes"`
	Warnings []string     `json:"Warnings"`
}

type volumeJSON struct {
	Name       string            `json:"Name"`
	Driver     string            `json:"Driver"`
	Mountpoint string            `json:"Mountpoint"`
	Scope      string            `json:"Scope"`
	CreatedAt  string            `json:"CreatedAt"`
	Labels     map[string]string `json:"Labels"`
}

func (fd *FakeDaemon) handleVolumeList(w http.ResponseWriter, r *http.Request) {
	names := fd.data.SortedVolumes()
	volumes := make([]volumeJSON, 0, len(names))

	for _, name := range names {
		volumes = append(volumes, volumeJSON{
			Name:       name,
			Driver:     "local",
			Mountpoint: fmt.Sprintf("/var/lib/docker/volumes/%s/_data", name),
			Scope:      "local",
			CreatedAt:  "2026-01-01T00:00:00Z",
			Labels:     map[string]string{},
		})
	}

	writeJSON(w, http.StatusOK, volumeListJSON{
		Volumes:  volumes,
		Warnings: []string{},
	})
}

func (fd *FakeDaemon) handleVolumeInspect(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	if _, ok := fd.data.volumes[name]; !ok {
		http.Error(w, fmt.Sprintf(`{"message":"volume %s not found"}`, name), http.StatusNotFound)
		return
	}

	resp := volumeJSON{
		Name:       name,
		Driver:     "local",
		Mountpoint: fmt.Sprintf("/var/lib/docker/volumes/%s/_data", name),
		Scope:      "local",
		CreatedAt:  "2026-01-01T00:00:00Z",
		Labels:     map[string]string{},
	}

	writeJSON(w, http.StatusOK, resp)
}

// --- Events ---

func (fd *FakeDaemon) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Subscribe to events
	subID, ch := fd.subscribeEvents()
	defer fd.unsubscribeEvents(subID)

	enc := json.NewEncoder(w)
	for {
		select {
		case <-r.Context().Done():
			return
		case evt := <-ch:
			if err := enc.Encode(evt); err != nil {
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

func (fd *FakeDaemon) subscribeEvents() (int, chan eventMessage) {
	fd.eventsMu.Lock()
	defer fd.eventsMu.Unlock()
	id := fd.nextSubID
	fd.nextSubID++
	ch := make(chan eventMessage, 64)
	fd.eventSubs[id] = ch
	return id, ch
}

func (fd *FakeDaemon) unsubscribeEvents(id int) {
	fd.eventsMu.Lock()
	defer fd.eventsMu.Unlock()
	if ch, ok := fd.eventSubs[id]; ok {
		close(ch)
		delete(fd.eventSubs, id)
	}
}

// publishEvent sends an event to all subscribers (non-blocking).
func (fd *FakeDaemon) publishEvent(action, containerID, project, service string) {
	fd.eventsMu.Lock()
	defer fd.eventsMu.Unlock()

	now := time.Now()
	evt := eventMessage{
		Status: action,
		ID:     containerID,
		Type:   "container",
		Action: action,
		Actor: eventActor{
			ID: containerID,
			Attributes: map[string]string{
				"com.docker.compose.project": project,
				"com.docker.compose.service": service,
			},
		},
		Time:     now.Unix(),
		TimeNano: now.UnixNano(),
	}

	for _, ch := range fd.eventSubs {
		select {
		case ch <- evt:
		default:
			// Drop if subscriber is slow
		}
	}
}

// --- Custom Mock State Endpoints ---

func (fd *FakeDaemon) handleMockServiceStateSet(w http.ResponseWriter, r *http.Request) {
	stack := r.PathValue("stack")
	service := r.PathValue("service")

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	fd.state.SetService(stack, service, body.Status)
	fd.world.Reset() // Rebuild world after state change

	containerID := fmt.Sprintf("mock-%s-%s-1", stack, service)
	switch body.Status {
	case "running":
		fd.publishEvent("start", containerID, stack, service)
	case "exited":
		fd.publishEvent("die", containerID, stack, service)
	case "paused":
		fd.publishEvent("pause", containerID, stack, service)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (fd *FakeDaemon) handleMockStateSet(w http.ResponseWriter, r *http.Request) {
	stack := r.PathValue("stack")

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	oldStatus := fd.state.Get(stack)
	fd.state.Set(stack, body.Status)
	fd.world.Reset() // Rebuild world after state change

	// Publish events for state transitions
	services := fd.getServices(stack)
	if oldStatus != body.Status {
		for _, svc := range services {
			containerID := fmt.Sprintf("mock-%s-%s-1", stack, svc)
			switch body.Status {
			case "running":
				fd.publishEvent("start", containerID, stack, svc)
			case "exited":
				fd.publishEvent("die", containerID, stack, svc)
			case "paused":
				fd.publishEvent("pause", containerID, stack, svc)
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (fd *FakeDaemon) handleMockStateDelete(w http.ResponseWriter, r *http.Request) {
	stack := r.PathValue("stack")

	// Publish destroy events before removing
	services := fd.getServices(stack)
	for _, svc := range services {
		containerID := fmt.Sprintf("mock-%s-%s-1", stack, svc)
		fd.publishEvent("destroy", containerID, stack, svc)
	}

	fd.state.Remove(stack)
	fd.world.Reset() // Rebuild world after state change
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (fd *FakeDaemon) handleMockReset(w http.ResponseWriter, r *http.Request) {
	fd.state.Reset()

	// Restore stacks directory from pristine source (if configured and
	// source differs from working dir — when they're the same, file
	// restoration would wipe the stacks).
	if fd.stacksSource != "" && !samePath(fd.stacksSource, fd.stacksDir) {
		if info, err := os.Stat(fd.stacksSource); err == nil && info.IsDir() {
			// Clear contents instead of removing the directory itself so any
			// fsnotify watcher (which watches this inode) stays valid.
			if err := ClearDirContents(fd.stacksDir); err != nil {
				slog.Error("mock reset: clear stacks dir", "err", err)
			}
			if err := CopyDirRecursive(fd.stacksSource, fd.stacksDir); err != nil {
				slog.Error("mock reset: copy stacks dir", "err", err)
			}
		}
	}

	// Rebuild mock data from (restored) stacks and reset world.
	fd.data = BuildMockData(fd.stacksDir)
	fd.state = DefaultDevStateFromData(fd.data)
	fd.world = BuildMockWorld(fd.data, fd.state, fd.stacksDir)

	// Return updateFlags so the caller can seed BoltDB image updates.
	resp := struct {
		OK          bool            `json:"ok"`
		UpdateFlags map[string]bool `json:"updateFlags,omitempty"`
	}{
		OK:          true,
		UpdateFlags: fd.data.UpdateFlags(),
	}
	writeJSON(w, http.StatusOK, resp)
}

// mockLogsJSON is the JSON response for /_mock/logs/{stack}/{service}.
type mockLogsJSON struct {
	BaseTime  string   `json:"base_time"`
	Startup   []string `json:"startup"`
	Heartbeat []string `json:"heartbeat"`
	Interval  string   `json:"interval"`
	Shutdown  []string `json:"shutdown"`
}

func (fd *FakeDaemon) handleMockLogs(w http.ResponseWriter, r *http.Request) {
	stack := r.PathValue("stack")
	service := r.PathValue("service")

	logs := fd.data.GetServiceLogs(stack, service)
	imageBase := "unknown"
	if img, ok := fd.data.serviceImages[stack+"/"+service]; ok {
		imageBase = extractImageBase(img)
	}

	// Expand startup and shutdown lines
	startup := make([]string, len(logs.Startup))
	for i, line := range logs.Startup {
		startup[i] = ExpandLogTemplate(line, i, logs.BaseTime, logs.Interval, imageBase)
	}
	shutdown := make([]string, len(logs.Shutdown))
	for i, line := range logs.Shutdown {
		shutdown[i] = ExpandLogTemplate(line, i, logs.BaseTime, logs.Interval, imageBase)
	}

	resp := mockLogsJSON{
		BaseTime:  logs.BaseTime.Format(time.RFC3339Nano),
		Startup:   startup,
		Heartbeat: logs.Heartbeat, // heartbeat lines stay as templates (expanded per-tick)
		Interval:  logs.Interval.String(),
		Shutdown:  shutdown,
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- Helpers ---

func (fd *FakeDaemon) getServices(stackName string) []string {
	prefix := stackName + "/"
	var services []string
	for key := range fd.data.serviceImages {
		if strings.HasPrefix(key, prefix) {
			svc := strings.TrimPrefix(key, prefix)
			services = append(services, svc)
		}
	}
	sort.Strings(services)

	// Fallback: parse compose file if MockData doesn't know about this stack
	if len(services) == 0 {
		composeFile := findComposeFilePath(filepath.Join(fd.stacksDir, stackName))
		if composeFile != "" {
			cd := parseComposeForMock(composeFile)
			for _, svc := range cd.services {
				services = append(services, svc.name)
			}
		}
	}
	return services
}

// StartEventsPoller starts a goroutine that polls MockState diffs and publishes
// events. This catches state changes made outside the /_mock/state endpoints
// (e.g., direct MockState.Set calls during tests).
func (fd *FakeDaemon) StartEventsPoller(ctx context.Context) {
	go func() {
		prev := fd.state.All()
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				curr := fd.state.All()
				for stack, status := range curr {
					oldStatus, existed := prev[stack]
					if !existed || oldStatus != status {
						services := fd.getServices(stack)
						for _, svc := range services {
							containerID := fmt.Sprintf("mock-%s-%s-1", stack, svc)
							switch status {
							case "running":
								fd.publishEvent("start", containerID, stack, svc)
							case "exited":
								fd.publishEvent("die", containerID, stack, svc)
							}
						}
					}
				}
				for stack := range prev {
					if _, exists := curr[stack]; !exists {
						services := fd.getServices(stack)
						for _, svc := range services {
							containerID := fmt.Sprintf("mock-%s-%s-1", stack, svc)
							fd.publishEvent("destroy", containerID, stack, svc)
						}
					}
				}
				prev = curr
			}
		}
	}()
}

// extractProjectFilter extracts the compose project name from a Docker API
// label filter value. The SDK may send either array form ["key=val"] or
// map form {"key=val":true}.
func extractProjectFilter(raw json.RawMessage) string {
	const prefix = "com.docker.compose.project="

	// Try array form first: ["com.docker.compose.project=name"]
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		for _, lbl := range arr {
			if after, ok := strings.CutPrefix(lbl, prefix); ok {
				return after
			}
		}
		return ""
	}

	// Try map form: {"com.docker.compose.project=name":true}
	var m map[string]bool
	if err := json.Unmarshal(raw, &m); err == nil {
		for lbl := range m {
			if after, ok := strings.CutPrefix(lbl, prefix); ok {
				return after
			}
		}
	}

	return ""
}

// parseSizeToBytes converts a human-readable size string like "245.3MiB" or "1.5GiB" to bytes.
func parseSizeToBytes(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "0B" || s == "" {
		return 0
	}

	var multiplier float64 = 1
	numStr := s

	if strings.HasSuffix(s, "GiB") {
		multiplier = 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(s, "GiB")
	} else if strings.HasSuffix(s, "MiB") {
		multiplier = 1024 * 1024
		numStr = strings.TrimSuffix(s, "MiB")
	} else if strings.HasSuffix(s, "KiB") {
		multiplier = 1024
		numStr = strings.TrimSuffix(s, "KiB")
	} else if strings.HasSuffix(s, "B") {
		numStr = strings.TrimSuffix(s, "B")
	}

	var val float64
	fmt.Sscanf(numStr, "%f", &val)
	return int64(val * multiplier)
}

// splitImageRef splits an image reference into repo and tag.
func splitImageRef(ref string) (string, string) {
	if idx := strings.Index(ref, "@"); idx >= 0 {
		return ref[:idx], "latest"
	}
	if idx := strings.LastIndex(ref, ":"); idx >= 0 {
		return ref[:idx], ref[idx+1:]
	}
	return ref, "latest"
}

// mockHash generates a deterministic 32-char hex hash from a string.
func mockHash(s string) string {
	var h uint64 = 14695981039346656037 // FNV offset basis
	for _, c := range s {
		h ^= uint64(c)
		h *= 1099511628211 // FNV prime
	}
	return fmt.Sprintf("%016x%016x", h, h^0xdeadbeefcafebabe)
}

// generateLayers creates mock image layers for image detail responses.
func generateLayers(imageRef, topID, created string) []docker.ImageLayer {
	h := simpleHash(imageRef)
	numLayers := 2 + int(h%3)

	layers := make([]docker.ImageLayer, 0, numLayers)

	cmd := "CMD [\"/bin/sh\"]"
	baseName := imageRef
	if idx := strings.LastIndex(baseName, "/"); idx >= 0 {
		baseName = baseName[idx+1:]
	}
	if idx := strings.Index(baseName, ":"); idx >= 0 {
		baseName = baseName[:idx]
	}
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

	layers = append(layers, docker.ImageLayer{
		ID:      topID,
		Created: created,
		Size:    "0B",
		Command: cmd,
	})

	for i := 1; i < numLayers-1; i++ {
		layerSize := fmt.Sprintf("%.1fMiB", float64(1+h%200)+float64(i)*10)
		layers = append(layers, docker.ImageLayer{
			ID:      "<missing>",
			Created: created,
			Size:    layerSize,
			Command: "RUN /bin/sh -c set -x && install dependencies # buildkit",
		})
	}

	baseSize := fmt.Sprintf("%.1fMiB", float64(5+h%500))
	layers = append(layers, docker.ImageLayer{
		ID:      "<missing>",
		Created: created,
		Size:    baseSize,
		Command: "/bin/sh -c #(nop) ADD file:... in /",
	})

	return layers
}
