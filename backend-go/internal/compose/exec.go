package compose

import (
    "bytes"
    "context"
    "fmt"
    "io"
    "log/slog"
    "os/exec"
    "path/filepath"
    "strings"
)

// Exec runs docker compose commands by shelling out to the CLI.
type Exec struct {
    StacksDir string
}

// Ls runs `docker compose ls --all --format json` and returns the raw JSON output.
func (e *Exec) Ls(ctx context.Context) ([]byte, error) {
    cmd := exec.CommandContext(ctx, "docker", "compose", "ls", "--all", "--format", "json")
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("compose ls: %w: %s", err, stderr.String())
    }
    return stdout.Bytes(), nil
}

// Up runs `docker compose up -d` for the given stack.
// Output is streamed to the writer. Returns when the command completes.
func (e *Exec) Up(ctx context.Context, stackName string, w io.Writer) error {
    return e.run(ctx, stackName, w, "up", "-d")
}

// Stop runs `docker compose stop` for the given stack.
func (e *Exec) Stop(ctx context.Context, stackName string, w io.Writer) error {
    return e.run(ctx, stackName, w, "stop")
}

// Down runs `docker compose down` for the given stack.
func (e *Exec) Down(ctx context.Context, stackName string, w io.Writer) error {
    return e.run(ctx, stackName, w, "down")
}

// Restart runs `docker compose restart` for the given stack.
func (e *Exec) Restart(ctx context.Context, stackName string, w io.Writer) error {
    return e.run(ctx, stackName, w, "restart")
}

// Pause runs `docker compose pause` for the given stack.
func (e *Exec) Pause(ctx context.Context, stackName string, w io.Writer) error {
    return e.run(ctx, stackName, w, "pause")
}

// Unpause runs `docker compose unpause` for the given stack.
func (e *Exec) Unpause(ctx context.Context, stackName string, w io.Writer) error {
    return e.run(ctx, stackName, w, "unpause")
}

// Pull runs `docker compose pull` for the given stack.
func (e *Exec) Pull(ctx context.Context, stackName string, w io.Writer) error {
    return e.run(ctx, stackName, w, "pull")
}

// PullAndUp runs `docker compose pull` then `docker compose up -d`.
func (e *Exec) PullAndUp(ctx context.Context, stackName string, w io.Writer) error {
    if err := e.Pull(ctx, stackName, w); err != nil {
        return err
    }
    return e.Up(ctx, stackName, w)
}

// Config runs `docker compose config --dry-run` for YAML validation.
func (e *Exec) Config(ctx context.Context, stackName string) ([]byte, error) {
    dir := filepath.Join(e.StacksDir, stackName)
    cmd := exec.CommandContext(ctx, "docker", "compose", "config", "--dry-run")
    cmd.Dir = dir
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("%s", strings.TrimSpace(stderr.String()))
    }
    return stdout.Bytes(), nil
}

// Ps runs `docker compose ps --format json` and returns the raw JSON.
func (e *Exec) Ps(ctx context.Context, stackName string) ([]byte, error) {
    dir := filepath.Join(e.StacksDir, stackName)
    cmd := exec.CommandContext(ctx, "docker", "compose", "ps", "--format", "json")
    cmd.Dir = dir
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("compose ps: %w: %s", err, stderr.String())
    }
    return stdout.Bytes(), nil
}

// ServiceUp starts a single service: `docker compose up -d <service>`
func (e *Exec) ServiceUp(ctx context.Context, stackName, serviceName string, w io.Writer) error {
    return e.run(ctx, stackName, w, "up", "-d", serviceName)
}

// ServiceStop stops a single service: `docker compose stop <service>`
func (e *Exec) ServiceStop(ctx context.Context, stackName, serviceName string, w io.Writer) error {
    return e.run(ctx, stackName, w, "stop", serviceName)
}

// ServiceRestart restarts a single service: `docker compose restart <service>`
func (e *Exec) ServiceRestart(ctx context.Context, stackName, serviceName string, w io.Writer) error {
    return e.run(ctx, stackName, w, "restart", serviceName)
}

// ServicePullAndUp pulls and restarts a single service.
func (e *Exec) ServicePullAndUp(ctx context.Context, stackName, serviceName string, w io.Writer) error {
    dir := filepath.Join(e.StacksDir, stackName)
    // Pull just the one service
    pullCmd := exec.CommandContext(ctx, "docker", "compose", "pull", serviceName)
    pullCmd.Dir = dir
    pullCmd.Stdout = w
    pullCmd.Stderr = w
    if err := pullCmd.Run(); err != nil {
        return fmt.Errorf("pull %s: %w", serviceName, err)
    }
    return e.ServiceUp(ctx, stackName, serviceName, w)
}

// Logs runs `docker compose logs -f` and streams output to the writer.
// The command runs until the context is cancelled.
func (e *Exec) Logs(ctx context.Context, stackName string, w io.Writer) error {
    dir := filepath.Join(e.StacksDir, stackName)
    cmd := exec.CommandContext(ctx, "docker", "compose", "logs", "-f", "--tail", "100")
    cmd.Dir = dir
    cmd.Stdout = w
    cmd.Stderr = w

    if err := cmd.Start(); err != nil {
        return fmt.Errorf("compose logs: %w", err)
    }
    return cmd.Wait()
}

// ServiceLogs streams logs for a single service.
func (e *Exec) ServiceLogs(ctx context.Context, stackName, serviceName string, w io.Writer) error {
    dir := filepath.Join(e.StacksDir, stackName)
    cmd := exec.CommandContext(ctx, "docker", "compose", "logs", "-f", "--tail", "100", serviceName)
    cmd.Dir = dir
    cmd.Stdout = w
    cmd.Stderr = w

    if err := cmd.Start(); err != nil {
        return fmt.Errorf("compose logs %s: %w", serviceName, err)
    }
    return cmd.Wait()
}

// run executes a docker compose command with output streaming.
func (e *Exec) run(ctx context.Context, stackName string, w io.Writer, composeArgs ...string) error {
    dir := filepath.Join(e.StacksDir, stackName)
    args := append([]string{"compose"}, composeArgs...)
    cmd := exec.CommandContext(ctx, "docker", args...)
    cmd.Dir = dir
    cmd.Stdout = w
    cmd.Stderr = w

    slog.Debug("compose exec", "stack", stackName, "args", composeArgs)

    if err := cmd.Run(); err != nil {
        return fmt.Errorf("docker compose %s: %w", strings.Join(composeArgs, " "), err)
    }
    return nil
}
