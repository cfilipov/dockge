package docker

import (
    "context"
    "io"
    "time"
)

// Client abstracts Docker daemon queries (reads only).
// Write operations (up, down, stop, restart, pull) remain as CLI shell-outs
// in the compose package.
type Client interface {
    // ContainerList returns containers, optionally filtered by compose project.
    // If all is true, includes stopped containers. If projectFilter is non-empty,
    // only returns containers belonging to that compose project.
    ContainerList(ctx context.Context, all bool, projectFilter string) ([]Container, error)

    // ContainerInspect returns the raw JSON inspect output for a container.
    ContainerInspect(ctx context.Context, id string) (string, error)

    // ContainerStats returns resource usage stats for running containers.
    // If projectFilter is non-empty, only returns stats for that compose project.
    ContainerStats(ctx context.Context, projectFilter string) (map[string]ContainerStat, error)

    // ContainerStartedAt returns when the container was last started.
    // Returns zero time if the container has never started or info is unavailable.
    ContainerStartedAt(ctx context.Context, containerID string) (time.Time, error)

    // ContainerLogs opens a log stream for a container.
    // Returns the stream, whether the container uses a TTY, and any error.
    // The caller must close the returned ReadCloser.
    ContainerLogs(ctx context.Context, containerID string, tail string, follow bool) (io.ReadCloser, bool, error)

    // ImageInspect returns the RepoDigests for a local image.
    // Returns nil if the image is not found locally.
    ImageInspect(ctx context.Context, imageRef string) ([]string, error)

    // DistributionInspect returns the remote (registry) digest for an image
    // without pulling it. Returns "" if unavailable.
    DistributionInspect(ctx context.Context, imageRef string) (string, error)

    // ContainerTop returns the running processes inside a container.
    // Returns column titles and a list of rows (each row is a list of values).
    ContainerTop(ctx context.Context, id string) ([]string, [][]string, error)

    // NetworkList returns summary info for all Docker networks.
    NetworkList(ctx context.Context) ([]NetworkSummary, error)

    // NetworkInspect returns detailed info for a single Docker network.
    NetworkInspect(ctx context.Context, networkID string) (*NetworkDetail, error)

    // ImageList returns summary info for all Docker images.
    ImageList(ctx context.Context) ([]ImageSummary, error)

    // ImageInspectDetail returns detailed info for a single Docker image,
    // including layers and containers using it.
    ImageInspectDetail(ctx context.Context, imageRef string) (*ImageDetail, error)

    // ImagePrune removes unused images. Returns human-readable reclaimed space string.
    ImagePrune(ctx context.Context, all bool) (string, error)

    // VolumeList returns summary info for all Docker volumes.
    VolumeList(ctx context.Context) ([]VolumeSummary, error)

    // VolumeInspect returns detailed info for a single Docker volume.
    VolumeInspect(ctx context.Context, volumeName string) (*VolumeDetail, error)

    // Events returns a channel of container lifecycle events and an error channel.
    // The channels are closed when the context is cancelled.
    Events(ctx context.Context) (<-chan ContainerEvent, <-chan error)

    // Close releases any resources held by the client.
    Close() error
}

// NewClient creates a Docker client. If mock is true, returns a MockClient
// that synthesizes container data in memory from compose.yaml files on disk
// (no Docker daemon or mock script needed). Otherwise returns an SDKClient
// that talks directly to the Docker daemon socket.
// The state and data parameters are only used when mock is true and may be nil otherwise.
func NewClient(mock bool, stacksDir string, state *MockState, data *MockData) (Client, error) {
    if mock {
        if data != nil {
            return NewMockClientWithData(stacksDir, state, data), nil
        }
        return NewMockClient(stacksDir, state), nil
    }
    return NewSDKClient()
}
