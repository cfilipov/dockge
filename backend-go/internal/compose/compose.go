package compose

import (
    "context"
    "io"
)

// Composer handles docker compose lifecycle commands.
// Read-only queries are handled by the docker.Client interface instead.
type Composer interface {
    // RunCompose executes a `docker compose` subcommand for a stack.
    // args are the compose args, e.g. ["up", "-d", "--remove-orphans"].
    RunCompose(ctx context.Context, stackName string, w io.Writer, args ...string) error

    // RunDocker executes a `docker` command in a stack directory.
    // args are the full docker args, e.g. ["image", "prune", "--all", "--force"].
    RunDocker(ctx context.Context, stackName string, w io.Writer, args ...string) error

    // Config runs `docker compose config --dry-run` for validation.
    Config(ctx context.Context, stackName string, w io.Writer) error

    // DownRemoveOrphans runs `docker compose down --remove-orphans`.
    DownRemoveOrphans(ctx context.Context, stackName string, w io.Writer) error

    // DownVolumes runs `docker compose down -v --remove-orphans`.
    DownVolumes(ctx context.Context, stackName string, w io.Writer) error

    // ServiceUp starts a single service.
    ServiceUp(ctx context.Context, stackName, serviceName string, w io.Writer) error

    // ServiceStop stops a single service.
    ServiceStop(ctx context.Context, stackName, serviceName string, w io.Writer) error

    // ServiceRestart restarts a single service.
    ServiceRestart(ctx context.Context, stackName, serviceName string, w io.Writer) error

    // ServicePullAndUp pulls and restarts a single service.
    ServicePullAndUp(ctx context.Context, stackName, serviceName string, w io.Writer) error
}
