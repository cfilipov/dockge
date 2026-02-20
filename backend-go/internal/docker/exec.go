package docker

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "os/exec"
    "strings"
)

// Stats runs `docker stats --no-stream --format json` and returns parsed stats
// keyed by container name.
func Stats(ctx context.Context) (map[string]interface{}, error) {
    cmd := exec.CommandContext(ctx, "docker", "stats", "--no-stream", "--format", "{{json .}}")
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("docker stats: %w: %s", err, stderr.String())
    }

    result := make(map[string]interface{})
    for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
        if line == "" {
            continue
        }
        var stat map[string]interface{}
        if err := json.Unmarshal([]byte(line), &stat); err != nil {
            continue
        }
        if name, ok := stat["Name"].(string); ok {
            result[name] = stat
        }
    }
    return result, nil
}

// Inspect runs `docker inspect <containerName>` and returns the raw JSON string.
func Inspect(ctx context.Context, containerName string) (string, error) {
    cmd := exec.CommandContext(ctx, "docker", "inspect", containerName)
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("docker inspect: %w: %s", err, stderr.String())
    }
    return stdout.String(), nil
}

// NetworkList runs `docker network ls --format {{.Name}}` and returns network names.
func NetworkList(ctx context.Context) ([]string, error) {
    cmd := exec.CommandContext(ctx, "docker", "network", "ls", "--format", "{{.Name}}")
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("docker network ls: %w: %s", err, stderr.String())
    }

    var names []string
    for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
        if line != "" {
            names = append(names, line)
        }
    }
    return names, nil
}
