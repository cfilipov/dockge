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

// ImageDigest runs `docker image inspect <image>` and extracts the first RepoDigest.
// Returns "" if the image is not found locally or has no digest.
func ImageDigest(ctx context.Context, imageRef string) string {
    cmd := exec.CommandContext(ctx, "docker", "image", "inspect", imageRef, "--format", "{{json .RepoDigests}}")
    var stdout bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = nil

    if err := cmd.Run(); err != nil {
        return ""
    }

    var digests []string
    if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &digests); err != nil || len(digests) == 0 {
        return ""
    }
    // RepoDigests are in the form "repo@sha256:abc..."
    // Extract just the digest part
    for _, d := range digests {
        if idx := strings.Index(d, "@"); idx >= 0 {
            return d[idx+1:]
        }
    }
    return digests[0]
}

// ManifestDigest runs `docker manifest inspect <image>` and extracts the config digest.
// This gives the remote (registry) digest without pulling the image.
// Returns "" if the command fails (e.g., experimental not enabled, no network).
func ManifestDigest(ctx context.Context, imageRef string) string {
    cmd := exec.CommandContext(ctx, "docker", "manifest", "inspect", imageRef)
    var stdout bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = nil

    if err := cmd.Run(); err != nil {
        return ""
    }

    // Parse the manifest to extract the config digest
    var manifest struct {
        Config struct {
            Digest string `json:"digest"`
        } `json:"config"`
        // For manifest lists, check the first manifest
        Manifests []struct {
            Digest string `json:"digest"`
        } `json:"manifests"`
    }
    if err := json.Unmarshal(stdout.Bytes(), &manifest); err != nil {
        return ""
    }
    if manifest.Config.Digest != "" {
        return manifest.Config.Digest
    }
    // Manifest list — return the first manifest's digest
    if len(manifest.Manifests) > 0 {
        return manifest.Manifests[0].Digest
    }
    return ""
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
