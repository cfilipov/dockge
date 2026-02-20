package docker

import (
    "context"
    "io"
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

    // ContainerStats returns resource usage stats for all running containers.
    ContainerStats(ctx context.Context) (map[string]ContainerStat, error)

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

    // NetworkList returns the names of all Docker networks.
    NetworkList(ctx context.Context) ([]string, error)

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
func NewClient(mock bool, stacksDir string) (Client, error) {
    if mock {
        return NewMockClient(stacksDir), nil
    }
    return NewSDKClient()
}
