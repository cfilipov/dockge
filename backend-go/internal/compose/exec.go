package compose

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Exec implements Composer by shelling out to the docker CLI.
type Exec struct {
	StacksDir string
}

// Ensure Exec implements Composer at compile time.
var _ Composer = (*Exec)(nil)

// GlobalEnvArgs returns --env-file flags to prepend to compose args
// when global.env exists in the stacks directory. If the stack also has
// a local .env, it is re-added explicitly (--env-file overrides the
// default .env loading). Returns nil when no global.env exists.
func GlobalEnvArgs(stacksDir, stackName string) []string {
	globalPath := filepath.Join(stacksDir, "global.env")
	if _, err := os.Stat(globalPath); err != nil {
		return nil
	}
	args := []string{"--env-file", "../global.env"}
	localEnv := filepath.Join(stacksDir, stackName, ".env")
	if _, err := os.Stat(localEnv); err == nil {
		args = append(args, "--env-file", "./.env")
	}
	return args
}

func (e *Exec) RunCompose(ctx context.Context, stackName string, w io.Writer, args ...string) error {
	return e.run(ctx, stackName, w, args...)
}

func (e *Exec) RunDocker(ctx context.Context, stackName string, w io.Writer, args ...string) error {
	dir := filepath.Join(e.StacksDir, stackName)
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = dir
	cmd.Stdout = w
	cmd.Stderr = w

	slog.Debug("docker exec", "stack", stackName, "args", args)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

func (e *Exec) Config(ctx context.Context, stackName string, w io.Writer) error {
	dir := filepath.Join(e.StacksDir, stackName)
	envArgs := GlobalEnvArgs(e.StacksDir, stackName)
	args := []string{"compose"}
	args = append(args, envArgs...)
	args = append(args, "config", "--dry-run")
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stdout = w
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(stderr.String()))
	}
	return nil
}

func (e *Exec) DownRemoveOrphans(ctx context.Context, stackName string, w io.Writer) error {
	return e.run(ctx, stackName, w, "down", "--remove-orphans")
}

func (e *Exec) DownVolumes(ctx context.Context, stackName string, w io.Writer) error {
	return e.run(ctx, stackName, w, "down", "-v", "--remove-orphans")
}

func (e *Exec) ServiceUp(ctx context.Context, stackName, serviceName string, w io.Writer) error {
	return e.run(ctx, stackName, w, "up", "-d", serviceName)
}

func (e *Exec) ServiceStop(ctx context.Context, stackName, serviceName string, w io.Writer) error {
	return e.run(ctx, stackName, w, "stop", serviceName)
}

func (e *Exec) ServiceRestart(ctx context.Context, stackName, serviceName string, w io.Writer) error {
	return e.run(ctx, stackName, w, "restart", serviceName)
}

func (e *Exec) ServicePullAndUp(ctx context.Context, stackName, serviceName string, w io.Writer) error {
	dir := filepath.Join(e.StacksDir, stackName)
	envArgs := GlobalEnvArgs(e.StacksDir, stackName)
	pullArgs := []string{"compose"}
	pullArgs = append(pullArgs, envArgs...)
	pullArgs = append(pullArgs, "pull", serviceName)
	pullCmd := exec.CommandContext(ctx, "docker", pullArgs...)
	pullCmd.Dir = dir
	pullCmd.Stdout = w
	pullCmd.Stderr = w
	if err := pullCmd.Run(); err != nil {
		return fmt.Errorf("pull %s: %w", serviceName, err)
	}
	return e.ServiceUp(ctx, stackName, serviceName, w)
}

// run executes a docker compose command with output streaming.
func (e *Exec) run(ctx context.Context, stackName string, w io.Writer, composeArgs ...string) error {
	dir := filepath.Join(e.StacksDir, stackName)
	envArgs := GlobalEnvArgs(e.StacksDir, stackName)
	args := []string{"compose"}
	args = append(args, envArgs...)
	args = append(args, composeArgs...)
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
