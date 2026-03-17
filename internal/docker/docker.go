package docker

import (
    "context"
    "encoding/json"
    "io"
    "time"
)

// Client abstracts Docker daemon queries (reads only).
// Write operations (up, down, stop, restart, pull) remain as CLI shell-outs
// via exec.Command("docker", ...).
type Client interface {
    // ContainerList returns containers, optionally filtered by compose project.
    // If all is true, includes stopped containers. If projectFilter is non-empty,
    // only returns containers belonging to that compose project.
    ContainerList(ctx context.Context, all bool, projectFilter string) ([]Container, error)

    // ContainerListDetailed returns enriched container data for broadcasting.
    // Includes networks, mounts, ports, and imageId for cross-store joins.
    ContainerListDetailed(ctx context.Context) ([]ContainerBroadcast, error)

    // ContainerListDetailedByID returns enriched data for a single container.
    // Uses Docker API id filter. Returns empty slice if not found.
    ContainerListDetailedByID(ctx context.Context, containerID string) ([]ContainerBroadcast, error)

    // ContainerInspect returns the raw JSON inspect output for a container.
    ContainerInspect(ctx context.Context, id string) (json.RawMessage, error)

    // ContainerStatStream opens a streaming stats connection for a single container.
    // Returns a channel that receives one ContainerStat per Docker stats frame.
    // The channel closes when ctx is cancelled or the stream ends.
    ContainerStatStream(ctx context.Context, containerName string) (<-chan ContainerStat, error)

    // ContainerStart starts a stopped container.
    // Only used in tests to transition mock containers from exited → running.
    ContainerStart(ctx context.Context, containerID string) error

    // ContainerStartedAt returns when the container was last started.
    // Returns zero time if the container has never started or info is unavailable.
    ContainerStartedAt(ctx context.Context, containerID string) (time.Time, error)

    // ContainerLogs opens a log stream for a container.
    // Returns the stream, whether the container uses a TTY, and any error.
    // The caller must close the returned ReadCloser.
    ContainerLogs(ctx context.Context, containerID string, tail string, follow bool, timestamps bool) (io.ReadCloser, bool, error)

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

    // NetworkListByID returns summary info for a single network by ID.
    // Returns empty slice if not found.
    NetworkListByID(ctx context.Context, networkID string) ([]NetworkSummary, error)

    // NetworkInspect returns detailed info for a single Docker network.
    NetworkInspect(ctx context.Context, networkID string) (*NetworkDetail, error)

    // ImageList returns summary info for all Docker images.
    ImageList(ctx context.Context) ([]ImageSummary, error)

    // ImageListByID returns summary info for a single image by ID.
    // Returns empty slice if not found.
    ImageListByID(ctx context.Context, imageID string) ([]ImageSummary, error)

    // ImageInspectDetail returns detailed info for a single Docker image,
    // including layers.
    ImageInspectDetail(ctx context.Context, imageRef string) (*ImageDetail, error)

    // ImagePrune removes unused images. Returns human-readable reclaimed space string.
    ImagePrune(ctx context.Context, all bool) (string, error)

    // VolumeList returns summary info for all Docker volumes.
    VolumeList(ctx context.Context) ([]VolumeSummary, error)

    // VolumeListByName returns summary info for a single volume by name.
    // Returns empty slice if not found.
    VolumeListByName(ctx context.Context, volumeName string) ([]VolumeSummary, error)

    // VolumeInspect returns detailed info for a single Docker volume.
    VolumeInspect(ctx context.Context, volumeName string) (*VolumeDetail, error)

    // Events returns a channel of Docker resource lifecycle events and an error channel.
    // Subscribes to container, network, image, and volume events.
    // The channels are closed when the context is cancelled.
    Events(ctx context.Context) (<-chan DockerEvent, <-chan error)

    // Close releases any resources held by the client.
    Close() error
}
