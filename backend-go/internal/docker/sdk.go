package docker

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "runtime/debug"
    "strconv"
    "strings"
    "sync"

    "time"

    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/api/types/events"
    "github.com/docker/docker/api/types/filters"
    "github.com/docker/docker/api/types/image"
    "github.com/docker/docker/api/types/network"
    "github.com/docker/docker/client"
    "github.com/docker/docker/pkg/stdcopy"
)

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

        health := ""
        if c.State == "running" && c.Status != "" {
            // Status contains health info like "(healthy)" or "(unhealthy)"
            lower := strings.ToLower(c.Status)
            if strings.Contains(lower, "(healthy)") {
                health = "healthy"
            } else if strings.Contains(lower, "(unhealthy)") {
                health = "unhealthy"
            } else if strings.Contains(lower, "(health: starting)") {
                health = "starting"
            }
        }

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

            var stats container.StatsResponse
            if err := json.NewDecoder(statsResp.Body).Decode(&stats); err != nil {
                statsResp.Body.Close()
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

            ch <- statResult{
                name: name,
                stat: ContainerStat{
                    Name:     name,
                    CPUPerc:  strconv.FormatFloat(cpuPerc, 'f', 2, 64) + "%",
                    MemPerc:  strconv.FormatFloat(memPerc, 'f', 2, 64) + "%",
                    MemUsage: formatBytes(memUsage) + " / " + formatBytes(memLimit),
                    NetIO:    formatBytes(netRx) + " / " + formatBytes(netTx),
                    BlockIO:  formatBytes(blkRead) + " / " + formatBytes(blkWrite),
                    PIDs:     strconv.FormatUint(stats.PidsStats.Current, 10),
                },
            }
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

    // Count containers per image ID
    containers, _ := s.cli.ContainerList(ctx, container.ListOptions{All: true})
    countByID := make(map[string]int, len(containers))
    for _, c := range containers {
        countByID[c.ImageID]++
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
            ID:         img.ID,
            RepoTags:   tags,
            Size:       formatBytes(uint64(img.Size)),
            Created:    time.Unix(img.Created, 0).UTC().Format(time.RFC3339),
            Containers: countByID[img.ID],
        })
    }
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

    // Find containers using this image
    allContainers, _ := s.cli.ContainerList(ctx, container.ListOptions{All: true})
    var imgContainers []ImageContainer
    for _, c := range allContainers {
        if c.ImageID == resp.ID || c.Image == imageRef {
            name := ""
            if len(c.Names) > 0 {
                name = strings.TrimPrefix(c.Names[0], "/")
            }
            imgContainers = append(imgContainers, ImageContainer{
                Name:        name,
                ContainerID: c.ID,
                State:       c.State,
            })
        }
    }
    if imgContainers == nil {
        imgContainers = []ImageContainer{}
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
        Containers:   imgContainers,
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

    // The Docker list API does not populate the Containers field —
    // only inspect does. Count containers per network from the
    // container list instead.
    containers, _ := s.cli.ContainerList(ctx, container.ListOptions{All: true})
    countByNet := make(map[string]int)
    for _, c := range containers {
        if c.NetworkSettings != nil {
            for netName := range c.NetworkSettings.Networks {
                countByNet[netName]++
            }
        }
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
            Containers: countByNet[n.Name],
        })
    }
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

func (s *SDKClient) Events(ctx context.Context) (<-chan ContainerEvent, <-chan error) {
    out := make(chan ContainerEvent, 64)
    outErr := make(chan error, 1)

    opts := events.ListOptions{
        Filters: filters.NewArgs(
            filters.Arg("type", string(events.ContainerEventType)),
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
                // Only process relevant actions
                switch msg.Action {
                case events.ActionStart, events.ActionStop, events.ActionDie,
                    events.ActionPause, events.ActionUnPause,
                    events.ActionDestroy, events.ActionCreate:
                    // ok
                default:
                    // Also handle health_status events
                    if !strings.HasPrefix(string(msg.Action), "health_status") {
                        continue
                    }
                }

                evt := ContainerEvent{
                    Action:      string(msg.Action),
                    ContainerID: msg.Actor.ID,
                    Project:     msg.Actor.Attributes["com.docker.compose.project"],
                    Service:     msg.Actor.Attributes["com.docker.compose.service"],
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

// Ensure SDKClient implements Client at compile time.
var _ Client = (*SDKClient)(nil)
