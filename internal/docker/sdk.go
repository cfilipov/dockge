package docker

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "runtime/debug"
    "sort"
    "strconv"
    "strings"
    "sync"

    "time"

    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/api/types/events"
    "github.com/docker/docker/api/types/filters"
    "github.com/docker/docker/api/types/image"
    "github.com/docker/docker/api/types/network"
    "github.com/docker/docker/api/types/volume"
    "github.com/docker/docker/client"
    "github.com/docker/docker/pkg/stdcopy"
)

// parseHealthFromStatus extracts the health status from Docker's human-readable
// Status string (e.g. "Up 2 hours (unhealthy)"). Returns "healthy", "unhealthy",
// "starting", or "" if no healthcheck is configured.
func parseHealthFromStatus(state, status string) string {
	if state != "running" || status == "" {
		return ""
	}
	lower := strings.ToLower(status)
	if strings.HasSuffix(lower, "(unhealthy)") {
		return "unhealthy"
	}
	if strings.HasSuffix(lower, "(healthy)") {
		return "healthy"
	}
	if strings.HasSuffix(lower, "(health: starting)") {
		return "starting"
	}
	return ""
}

// SDKClient implements Client using the Docker Engine SDK.
type SDKClient struct {
    cli *client.Client
}

// NewSDKClient creates an SDKClient that connects to the Docker daemon
// via the default socket (DOCKER_HOST or /var/run/docker.sock).
func NewSDKClient() (*SDKClient, error) {
    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
        return nil, fmt.Errorf("docker sdk: %w", err)
    }
    return &SDKClient{cli: cli}, nil
}

// NewSDKClientWithHost creates an SDKClient connected to a specific Docker host.
// The host parameter should be a full URI like "unix:///path/to/docker.sock".
func NewSDKClientWithHost(host string) (*SDKClient, error) {
    cli, err := client.NewClientWithOpts(client.WithHost(host), client.WithAPIVersionNegotiation())
    if err != nil {
        return nil, fmt.Errorf("docker sdk with host: %w", err)
    }
    return &SDKClient{cli: cli}, nil
}

func (s *SDKClient) ContainerList(ctx context.Context, all bool, projectFilter string) ([]Container, error) {
    opts := container.ListOptions{All: all}
    if projectFilter != "" {
        opts.Filters = filters.NewArgs(
            filters.Arg("label", "com.docker.compose.project="+projectFilter),
        )
    }

    raw, err := s.cli.ContainerList(ctx, opts)
    if err != nil {
        return nil, fmt.Errorf("container list: %w", err)
    }

    result := make([]Container, 0, len(raw))
    for _, c := range raw {
        name := ""
        if len(c.Names) > 0 {
            name = strings.TrimPrefix(c.Names[0], "/")
        }

        health := parseHealthFromStatus(c.State, c.Status)

        result = append(result, Container{
            ID:      c.ID,
            Name:    name,
            Project: c.Labels["com.docker.compose.project"],
            Service: c.Labels["com.docker.compose.service"],
            Image:   c.Image,
            State:   c.State,
            Health:  health,
        })
    }
    return result, nil
}

// ContainerBroadcastList returns enriched container data for the broadcast channel.
// Includes networks, mounts, ports, and imageId for cross-store joins.
func (s *SDKClient) ContainerBroadcastList(ctx context.Context) ([]ContainerBroadcast, error) {
    raw, err := s.cli.ContainerList(ctx, container.ListOptions{All: true})
    if err != nil {
        return nil, fmt.Errorf("container broadcast list: %w", err)
    }

    result := make([]ContainerBroadcast, 0, len(raw))
    for _, c := range raw {
        name := ""
        if len(c.Names) > 0 {
            name = strings.TrimPrefix(c.Names[0], "/")
        }

        health := parseHealthFromStatus(c.State, c.Status)

        // Extract network endpoints
        networks := make(map[string]ContainerNetwork)
        if c.NetworkSettings != nil {
            for netName, ep := range c.NetworkSettings.Networks {
                networks[netName] = ContainerNetwork{
                    IPv4: ep.IPAddress,
                    IPv6: ep.GlobalIPv6Address,
                    MAC:  ep.MacAddress,
                }
            }
        }

        // Extract mounts
        mounts := make([]ContainerMount, 0, len(c.Mounts))
        for _, m := range c.Mounts {
            mounts = append(mounts, ContainerMount{
                Name: m.Name,
                Type: string(m.Type),
            })
        }

        // Extract ports
        ports := make([]ContainerPort, 0, len(c.Ports))
        for _, p := range c.Ports {
            ports = append(ports, ContainerPort{
                HostPort:      p.PublicPort,
                ContainerPort: p.PrivatePort,
                Protocol:      p.Type,
            })
        }

        svc := c.Labels["com.docker.compose.service"]
        project := c.Labels["com.docker.compose.project"]

        result = append(result, ContainerBroadcast{
            Name:        name,
            ContainerID: c.ID,
            ServiceName: svc,
            StackName:   project,
            State:       strings.ToLower(c.State),
            Health:      strings.ToLower(health),
            Image:       c.Image,
            ImageID:     c.ImageID,
            Networks:    networks,
            Mounts:      mounts,
            Ports:       ports,
        })
    }

    // Sort by name for deterministic serialization
    sort.Slice(result, func(i, j int) bool {
        return result[i].Name < result[j].Name
    })

    return result, nil
}

func (s *SDKClient) ContainerInspect(ctx context.Context, id string) (string, error) {
    raw, err := s.cli.ContainerInspect(ctx, id)
    if err != nil {
        return "", fmt.Errorf("container inspect: %w", err)
    }
    // Return as JSON array (matching `docker inspect` CLI output)
    data, err := json.MarshalIndent([]interface{}{raw}, "", "  ")
    if err != nil {
        return "", fmt.Errorf("marshal inspect: %w", err)
    }
    return string(data), nil
}

func (s *SDKClient) ContainerStats(ctx context.Context, projectFilter string) (map[string]ContainerStat, error) {
    // List running containers, optionally filtered by compose project
    opts := container.ListOptions{}
    if projectFilter != "" {
        opts.Filters = filters.NewArgs(
            filters.Arg("label", "com.docker.compose.project="+projectFilter),
        )
    }
    containers, err := s.cli.ContainerList(ctx, opts)
    if err != nil {
        return nil, fmt.Errorf("container list for stats: %w", err)
    }

    // Fetch stats for all containers in parallel. Each Docker stats call
    // blocks ~1-2s waiting for a CPU delta sample, so serial fetching for
    // N containers takes N*1.5s. Parallel brings it down to ~1.5s total.
    // FreeOSMemory() at the end reclaims the brief memory spike.
    type statResult struct {
        name string
        stat ContainerStat
    }

    ch := make(chan statResult, len(containers))
    var wg sync.WaitGroup

    for _, c := range containers {
        c := c // capture loop variable
        wg.Add(1)
        go func() {
            defer wg.Done()

            name := ""
            if len(c.Names) > 0 {
                name = strings.TrimPrefix(c.Names[0], "/")
            }

            statsResp, err := s.cli.ContainerStats(ctx, c.ID, false)
            if err != nil {
                ch <- statResult{} // empty, will be skipped
                return
            }

            stats := statsResponsePool.Get().(*container.StatsResponse)
            if err := json.NewDecoder(statsResp.Body).Decode(stats); err != nil {
                statsResp.Body.Close()
                statsResponsePool.Put(stats)
                ch <- statResult{}
                return
            }
            statsResp.Body.Close()

            // Calculate CPU percentage
            cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
            systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
            cpuPerc := 0.0
            if systemDelta > 0 && cpuDelta > 0 {
                cpuPerc = (cpuDelta / systemDelta) * float64(stats.CPUStats.OnlineCPUs) * 100.0
            }

            // Memory usage
            memUsage := stats.MemoryStats.Usage - stats.MemoryStats.Stats["cache"]
            memLimit := stats.MemoryStats.Limit
            memPerc := 0.0
            if memLimit > 0 {
                memPerc = float64(memUsage) / float64(memLimit) * 100.0
            }

            // Network I/O
            var netRx, netTx uint64
            for _, v := range stats.Networks {
                netRx += v.RxBytes
                netTx += v.TxBytes
            }

            // Block I/O
            var blkRead, blkWrite uint64
            for _, bio := range stats.BlkioStats.IoServiceBytesRecursive {
                switch bio.Op {
                case "read", "Read":
                    blkRead += bio.Value
                case "write", "Write":
                    blkWrite += bio.Value
                }
            }

            // Build stat strings using AppendFloat to avoid intermediate allocations
            buf := make([]byte, 0, 16)
            buf = strconv.AppendFloat(buf, cpuPerc, 'f', 2, 64)
            buf = append(buf, '%')
            cpuStr := string(buf)

            buf = buf[:0]
            buf = strconv.AppendFloat(buf, memPerc, 'f', 2, 64)
            buf = append(buf, '%')
            memPercStr := string(buf)

            ch <- statResult{
                name: name,
                stat: ContainerStat{
                    Name:     name,
                    CPUPerc:  cpuStr,
                    MemPerc:  memPercStr,
                    MemUsage: formatBytesPair(memUsage, memLimit),
                    NetIO:    formatBytesPair(netRx, netTx),
                    BlockIO:  formatBytesPair(blkRead, blkWrite),
                    PIDs:     strconv.FormatUint(stats.PidsStats.Current, 10),
                },
            }

            // Zero and return stats to pool
            *stats = container.StatsResponse{}
            statsResponsePool.Put(stats)
        }()
    }

    // Close channel when all goroutines finish
    go func() {
        wg.Wait()
        close(ch)
    }()

    result := make(map[string]ContainerStat, len(containers))
    for r := range ch {
        if r.name != "" {
            result[r.name] = r.stat
        }
    }

    // Return memory to OS promptly — stats responses are large and
    // the Go runtime otherwise holds onto freed pages for minutes.
    debug.FreeOSMemory()

    return result, nil
}

func (s *SDKClient) ContainerTop(ctx context.Context, id string) ([]string, [][]string, error) {
    resp, err := s.cli.ContainerTop(ctx, id, []string{"-eo", "pid,user,args"})
    if err != nil {
        return nil, nil, fmt.Errorf("container top: %w", err)
    }
    return resp.Titles, resp.Processes, nil
}

func (s *SDKClient) ContainerStartedAt(ctx context.Context, containerID string) (time.Time, error) {
    inspect, err := s.cli.ContainerInspect(ctx, containerID)
    if err != nil {
        return time.Time{}, fmt.Errorf("inspect for started_at: %w", err)
    }
    if inspect.State == nil || inspect.State.StartedAt == "" {
        return time.Time{}, nil
    }
    t, err := time.Parse(time.RFC3339Nano, inspect.State.StartedAt)
    if err != nil {
        return time.Time{}, nil
    }
    return t, nil
}

func (s *SDKClient) ContainerLogs(ctx context.Context, containerID string, tail string, follow bool) (io.ReadCloser, bool, error) {
    // Check if container uses TTY
    inspect, err := s.cli.ContainerInspect(ctx, containerID)
    if err != nil {
        return nil, false, fmt.Errorf("inspect for logs: %w", err)
    }
    isTTY := inspect.Config.Tty

    opts := container.LogsOptions{
        ShowStdout: true,
        ShowStderr: true,
        Follow:     follow,
        Tail:       tail,
        Timestamps: false,
    }

    stream, err := s.cli.ContainerLogs(ctx, containerID, opts)
    if err != nil {
        return nil, false, fmt.Errorf("container logs: %w", err)
    }

    if isTTY {
        // TTY containers: raw stream, no multiplexing
        return stream, true, nil
    }

    // Non-TTY containers: Docker multiplexes stdout/stderr with 8-byte headers.
    // Demux using stdcopy into a pipe.
    pr, pw := io.Pipe()
    go func() {
        _, err := stdcopy.StdCopy(pw, pw, stream)
        stream.Close()
        pw.CloseWithError(err)
    }()

    return pr, false, nil
}

func (s *SDKClient) ImageInspect(ctx context.Context, imageRef string) ([]string, error) {
    resp, _, err := s.cli.ImageInspectWithRaw(ctx, imageRef)
    if err != nil {
        if client.IsErrNotFound(err) {
            return nil, nil
        }
        return nil, fmt.Errorf("image inspect: %w", err)
    }
    return resp.RepoDigests, nil
}

func (s *SDKClient) DistributionInspect(ctx context.Context, imageRef string) (string, error) {
    resp, err := s.cli.DistributionInspect(ctx, imageRef, "")
    if err != nil {
        // Not available (auth required, registry down, etc.) — not an error for our purposes
        return "", nil
    }
    return string(resp.Descriptor.Digest), nil
}

func (s *SDKClient) ImageList(ctx context.Context) ([]ImageSummary, error) {
    imgs, err := s.cli.ImageList(ctx, image.ListOptions{})
    if err != nil {
        return nil, fmt.Errorf("image list: %w", err)
    }

    result := make([]ImageSummary, 0, len(imgs))
    for _, img := range imgs {
        tags := make([]string, 0, len(img.RepoTags))
        for _, t := range img.RepoTags {
            if t != "<none>:<none>" {
                tags = append(tags, t)
            }
        }

        result = append(result, ImageSummary{
            ID:       img.ID,
            RepoTags: tags,
            Size:     formatBytes(uint64(img.Size)),
            Created:  time.Unix(img.Created, 0).UTC().Format(time.RFC3339),
            Dangling: len(tags) == 0,
        })
    }

    // Sort by ID for deterministic serialization
    sort.Slice(result, func(i, j int) bool {
        return result[i].ID < result[j].ID
    })

    return result, nil
}

func (s *SDKClient) ImageInspectDetail(ctx context.Context, imageRef string) (*ImageDetail, error) {
    resp, _, err := s.cli.ImageInspectWithRaw(ctx, imageRef)
    if err != nil {
        return nil, fmt.Errorf("image inspect detail: %w", err)
    }

    history, err := s.cli.ImageHistory(ctx, imageRef)
    if err != nil {
        return nil, fmt.Errorf("image history: %w", err)
    }

    layers := make([]ImageLayer, 0, len(history))
    for _, h := range history {
        id := "<missing>"
        if h.ID != "<missing>" && h.ID != "" {
            if len(h.ID) > 12 {
                id = h.ID[:12]
            } else {
                id = h.ID
            }
        }
        layers = append(layers, ImageLayer{
            ID:      id,
            Created: time.Unix(h.Created, 0).UTC().Format(time.RFC3339),
            Size:    formatBytes(uint64(h.Size)),
            Command: h.CreatedBy,
        })
    }

    tags := make([]string, 0, len(resp.RepoTags))
    for _, t := range resp.RepoTags {
        if t != "<none>:<none>" {
            tags = append(tags, t)
        }
    }

    workingDir := ""
    if resp.Config != nil {
        workingDir = resp.Config.WorkingDir
    }

    return &ImageDetail{
        ID:           resp.ID,
        RepoTags:     tags,
        Size:         formatBytes(uint64(resp.Size)),
        Created:      resp.Created,
        Architecture: resp.Architecture,
        OS:           resp.Os,
        WorkingDir:   workingDir,
        Layers:       layers,
    }, nil
}

func (s *SDKClient) ImagePrune(ctx context.Context, all bool) (string, error) {
    pruneFilters := filters.NewArgs()
    if !all {
        pruneFilters.Add("dangling", "true")
    }
    report, err := s.cli.ImagesPrune(ctx, pruneFilters)
    if err != nil {
        return "", fmt.Errorf("image prune: %w", err)
    }
    return "Total reclaimed space: " + formatBytes(report.SpaceReclaimed), nil
}

func (s *SDKClient) NetworkList(ctx context.Context) ([]NetworkSummary, error) {
    networks, err := s.cli.NetworkList(ctx, network.ListOptions{})
    if err != nil {
        return nil, fmt.Errorf("network list: %w", err)
    }

    result := make([]NetworkSummary, 0, len(networks))
    for _, n := range networks {
        result = append(result, NetworkSummary{
            Name:       n.Name,
            ID:         n.ID,
            Driver:     n.Driver,
            Scope:      n.Scope,
            Internal:   n.Internal,
            Attachable: n.Attachable,
            Ingress:    n.Ingress,
            Labels:     n.Labels,
        })
    }

    // Sort by name for deterministic serialization
    sort.Slice(result, func(i, j int) bool {
        return result[i].Name < result[j].Name
    })

    return result, nil
}

func (s *SDKClient) NetworkInspect(ctx context.Context, networkID string) (*NetworkDetail, error) {
    raw, err := s.cli.NetworkInspect(ctx, networkID, network.InspectOptions{})
    if err != nil {
        return nil, fmt.Errorf("network inspect: %w", err)
    }

    ipam := make([]NetworkIPAM, 0, len(raw.IPAM.Config))
    for _, cfg := range raw.IPAM.Config {
        ipam = append(ipam, NetworkIPAM{
            Subnet:  cfg.Subnet,
            Gateway: cfg.Gateway,
        })
    }

    containers := make([]NetworkContainerDetail, 0, len(raw.Containers))
    for id, ep := range raw.Containers {
        containers = append(containers, NetworkContainerDetail{
            Name:        ep.Name,
            ContainerID: id,
            IPv4:        ep.IPv4Address,
            IPv6:        ep.IPv6Address,
            MAC:         ep.MacAddress,
        })
    }
    sort.Slice(containers, func(i, j int) bool {
        return containers[i].Name < containers[j].Name
    })

    return &NetworkDetail{
        Name:       raw.Name,
        ID:         raw.ID,
        Driver:     raw.Driver,
        Scope:      raw.Scope,
        Internal:   raw.Internal,
        Attachable: raw.Attachable,
        Ingress:    raw.Ingress,
        IPv6:       raw.EnableIPv6,
        Created:    raw.Created.Format("2006-01-02T15:04:05Z"),
        IPAM:       ipam,
        Containers: containers,
    }, nil
}

func (s *SDKClient) VolumeList(ctx context.Context) ([]VolumeSummary, error) {
    volResp, err := s.cli.VolumeList(ctx, volume.ListOptions{})
    if err != nil {
        return nil, fmt.Errorf("volume list: %w", err)
    }

    result := make([]VolumeSummary, 0, len(volResp.Volumes))
    for _, v := range volResp.Volumes {
        result = append(result, VolumeSummary{
            Name:       v.Name,
            Driver:     v.Driver,
            Mountpoint: v.Mountpoint,
            Labels:     v.Labels,
        })
    }

    // Sort by name for deterministic serialization
    sort.Slice(result, func(i, j int) bool {
        return result[i].Name < result[j].Name
    })

    return result, nil
}

func (s *SDKClient) VolumeInspect(ctx context.Context, volumeName string) (*VolumeDetail, error) {
    raw, err := s.cli.VolumeInspect(ctx, volumeName)
    if err != nil {
        return nil, fmt.Errorf("volume inspect: %w", err)
    }

    return &VolumeDetail{
        Name:       raw.Name,
        Driver:     raw.Driver,
        Mountpoint: raw.Mountpoint,
        Scope:      raw.Scope,
        Created:    raw.CreatedAt,
    }, nil
}

func (s *SDKClient) Events(ctx context.Context) (<-chan DockerEvent, <-chan error) {
    out := make(chan DockerEvent, 64)
    outErr := make(chan error, 1)

    // Subscribe to container, network, image, and volume events
    opts := events.ListOptions{
        Filters: filters.NewArgs(
            filters.Arg("type", string(events.ContainerEventType)),
            filters.Arg("type", string(events.NetworkEventType)),
            filters.Arg("type", string(events.ImageEventType)),
            filters.Arg("type", string(events.VolumeEventType)),
        ),
    }

    msgCh, errCh := s.cli.Events(ctx, opts)

    go func() {
        defer close(out)
        defer close(outErr)

        for {
            select {
            case msg, ok := <-msgCh:
                if !ok {
                    return
                }

                evtType := string(msg.Type)
                action := string(msg.Action)

                // Filter to relevant actions per type
                switch msg.Type {
                case events.ContainerEventType:
                    switch msg.Action {
                    case events.ActionStart, events.ActionStop, events.ActionDie,
                        events.ActionPause, events.ActionUnPause,
                        events.ActionDestroy, events.ActionCreate:
                        // ok
                    default:
                        if !strings.HasPrefix(action, "health_status") {
                            continue
                        }
                    }
                case events.NetworkEventType:
                    // create, destroy, connect, disconnect
                case events.ImageEventType:
                    // pull, push, tag, untag, delete, build, import, load
                case events.VolumeEventType:
                    // create, destroy, mount, unmount
                default:
                    continue
                }

                evt := DockerEvent{
                    Type:   evtType,
                    Action: action,
                }
                // Container-specific fields
                if msg.Type == events.ContainerEventType {
                    evt.ContainerID = msg.Actor.ID
                    evt.Project = msg.Actor.Attributes["com.docker.compose.project"]
                    evt.Service = msg.Actor.Attributes["com.docker.compose.service"]
                }

                select {
                case out <- evt:
                case <-ctx.Done():
                    return
                }

            case err, ok := <-errCh:
                if !ok {
                    return
                }
                select {
                case outErr <- err:
                case <-ctx.Done():
                }
                return
            }
        }
    }()

    return out, outErr
}

func (s *SDKClient) Close() error {
    return s.cli.Close()
}

// statsResponsePool reuses container.StatsResponse structs to avoid
// repeated allocation of the ~2KB struct with nested maps.
var statsResponsePool = sync.Pool{
    New: func() any { return new(container.StatsResponse) },
}

// formatBytes formats a byte count as a human-readable string.
func formatBytes(b uint64) string {
    const unit = 1024
    if b < unit {
        return strconv.FormatUint(b, 10) + "B"
    }
    div, exp := uint64(unit), 0
    for n := b / unit; n >= unit; n /= unit {
        div *= unit
        exp++
    }
    return strconv.FormatFloat(float64(b)/float64(div), 'f', 1, 64) + string("KMGTPE"[exp]) + "iB"
}

// formatBytesPair formats two byte values as "a / b" using a strings.Builder
// to avoid intermediate string allocations from + concatenation.
func formatBytesPair(a, b uint64) string {
    var sb strings.Builder
    sb.Grow(32) // enough for two formatted values + " / "
    sb.WriteString(formatBytes(a))
    sb.WriteString(" / ")
    sb.WriteString(formatBytes(b))
    return sb.String()
}

// Ensure SDKClient implements Client at compile time.
var _ Client = (*SDKClient)(nil)
