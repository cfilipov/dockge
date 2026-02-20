package docker

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net"
    "net/http"
    "os"
    "runtime/debug"
    "strings"
    "sync"
    "time"

    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/api/types/events"
    "github.com/docker/docker/api/types/filters"
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
// The HTTP transport is tuned for low memory: small idle connection pool
// and short idle timeout so connections are released quickly.
func NewSDKClient() (*SDKClient, error) {
    // Determine socket path from DOCKER_HOST env, defaulting to the standard path.
    sockPath := "/var/run/docker.sock"
    if host, ok := os.LookupEnv("DOCKER_HOST"); ok && strings.HasPrefix(host, "unix://") {
        sockPath = strings.TrimPrefix(host, "unix://")
    }

    transport := &http.Transport{
        DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
            return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "unix", sockPath)
        },
        MaxIdleConns:        5,
        MaxIdleConnsPerHost: 3,
        IdleConnTimeout:     15 * time.Second,
        DisableKeepAlives:   false,
    }

    httpClient := &http.Client{Transport: transport}

    cli, err := client.NewClientWithOpts(
        client.FromEnv,
        client.WithAPIVersionNegotiation(),
        client.WithHTTPClient(httpClient),
    )
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

func (s *SDKClient) ContainerStats(ctx context.Context) (map[string]ContainerStat, error) {
    // List running containers first
    containers, err := s.cli.ContainerList(ctx, container.ListOptions{})
    if err != nil {
        return nil, fmt.Errorf("container list for stats: %w", err)
    }

    // Fetch stats with bounded concurrency. Each Docker stats call blocks
    // ~1-2s waiting for a CPU delta sample. We limit to 3 concurrent to
    // keep memory usage low (each goroutine holds an HTTP connection +
    // JSON decode buffer + StatsResponse struct).
    const maxConcurrent = 3

    type statResult struct {
        name string
        stat ContainerStat
    }

    ch := make(chan statResult, len(containers))
    sem := make(chan struct{}, maxConcurrent)
    var wg sync.WaitGroup

    for _, c := range containers {
        c := c // capture loop variable
        wg.Add(1)
        go func() {
            defer wg.Done()
            sem <- struct{}{}        // acquire
            defer func() { <-sem }() // release

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
                    CPUPerc:  fmt.Sprintf("%.2f%%", cpuPerc),
                    MemPerc:  fmt.Sprintf("%.2f%%", memPerc),
                    MemUsage: fmt.Sprintf("%s / %s", formatBytes(memUsage), formatBytes(memLimit)),
                    NetIO:    fmt.Sprintf("%s / %s", formatBytes(netRx), formatBytes(netTx)),
                    BlockIO:  fmt.Sprintf("%s / %s", formatBytes(blkRead), formatBytes(blkWrite)),
                    PIDs:     fmt.Sprintf("%d", stats.PidsStats.Current),
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

func (s *SDKClient) ImagePrune(ctx context.Context, all bool) (string, error) {
    pruneFilters := filters.NewArgs()
    if !all {
        pruneFilters.Add("dangling", "true")
    }
    report, err := s.cli.ImagesPrune(ctx, pruneFilters)
    if err != nil {
        return "", fmt.Errorf("image prune: %w", err)
    }
    return fmt.Sprintf("Total reclaimed space: %s", formatBytes(report.SpaceReclaimed)), nil
}

func (s *SDKClient) NetworkList(ctx context.Context) ([]string, error) {
    networks, err := s.cli.NetworkList(ctx, network.ListOptions{})
    if err != nil {
        return nil, fmt.Errorf("network list: %w", err)
    }

    names := make([]string, 0, len(networks))
    for _, n := range networks {
        names = append(names, n.Name)
    }
    return names, nil
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
        return fmt.Sprintf("%dB", b)
    }
    div, exp := uint64(unit), 0
    for n := b / unit; n >= unit; n /= unit {
        div *= unit
        exp++
    }
    return fmt.Sprintf("%.1f%ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// Ensure SDKClient implements Client at compile time.
var _ Client = (*SDKClient)(nil)
