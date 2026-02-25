package stack

import (
    "log/slog"
    "os"
    "path/filepath"
    "strings"

    "github.com/cfilipov/dockge/internal/docker"
)

// ContainerInfo holds the minimal container data needed for stack list building.
// This decouples the stack package from the full docker.Container type while
// allowing the same data to flow through.
type ContainerInfo = docker.Container

// IgnoreMap maps stackName → serviceName → true for services that should be
// excluded from status calculation (dockge.status.ignore=true).
type IgnoreMap map[string]map[string]bool

// GetStackListFromContainers scans the stacks directory and merges with container
// data from the Docker client. Containers are grouped by their compose project
// label to derive stack status. Services in ignoreServices are excluded from
// status counting.
func GetStackListFromContainers(stacksDir string, containers []ContainerInfo, ignoreServices ...IgnoreMap) map[string]*Stack {
    stacks := make(map[string]*Stack)

    // 1. Scan stacks directory for managed stacks
    entries, err := os.ReadDir(stacksDir)
    if err != nil {
        slog.Warn("scan stacks dir", "err", err, "dir", stacksDir)
        return stacks
    }

    for _, entry := range entries {
        if !entry.IsDir() {
            continue
        }
        name := entry.Name()

        // Skip if no compose file exists
        if !ComposeFileExists(stacksDir, name) {
            continue
        }

        s := &Stack{
            Name:              name,
            Status:            CREATED_FILE,
            IsManagedByDockge: true,
            Path:              filepath.Join(stacksDir, name),
        }

        // Detect compose file name
        for _, fname := range acceptedComposeFileNames {
            if _, err := os.Stat(filepath.Join(s.Path, fname)); err == nil {
                s.ComposeFileName = fname
                break
            }
        }

        // Detect override file name
        for _, fname := range acceptedComposeOverrideFileNames {
            if _, err := os.Stat(filepath.Join(s.Path, fname)); err == nil {
                s.ComposeOverrideFileName = fname
                break
            }
        }

        stacks[name] = s
    }

    // 2. Group containers by project and derive status
    if len(containers) == 0 {
        return stacks
    }

    // Build the ignore lookup (optional parameter)
    var ignore IgnoreMap
    if len(ignoreServices) > 0 && ignoreServices[0] != nil {
        ignore = ignoreServices[0]
    }

    // Group by compose project label
    type projectState struct {
        running   int
        exited    int
        created   int
        paused    int
        unhealthy int
    }
    projects := make(map[string]*projectState)

    for _, c := range containers {
        project := c.Project
        if project == "" {
            continue
        }

        // Skip status-ignored services
        if ignore != nil {
            svc := c.Service
            if svc == "" {
                svc = extractServiceFromName(c.Name)
            }
            if ignore[project] != nil && ignore[project][svc] {
                continue
            }
        }

        ps, ok := projects[project]
        if !ok {
            ps = &projectState{}
            projects[project] = ps
        }

        // Health field takes priority over State for status classification
        if strings.ToLower(c.Health) == "unhealthy" {
            ps.unhealthy++
        } else {
            switch strings.ToLower(c.State) {
            case "running":
                ps.running++
            case "exited", "dead":
                ps.exited++
            case "created":
                ps.created++
            case "paused":
                ps.paused++
            }
        }
    }

    for project, ps := range projects {
        s, exists := stacks[project]
        if !exists {
            // External stack (not in our managed directory)
            if project == "dockge" {
                continue // skip the dockge stack itself
            }
            s = &Stack{
                Name:              project,
                IsManagedByDockge: false,
            }
            stacks[project] = s
        }

        // Derive status from container states
        if ps.unhealthy > 0 {
            s.Status = UNHEALTHY
        } else if ps.running > 0 && ps.exited > 0 {
            s.Status = RUNNING_AND_EXITED
        } else if ps.running > 0 {
            s.Status = RUNNING
        } else if ps.exited > 0 {
            s.Status = EXITED
        } else if ps.created > 0 {
            s.Status = CREATED_STACK
        } else if ps.paused > 0 {
            s.Status = RUNNING // paused counts as running for UI purposes
        }
    }

    return stacks
}

// extractServiceFromName extracts the service name from a Docker Compose container name.
// Format: stackname-servicename-N (e.g., "web-app-nginx-1" -> "nginx").
// Best-effort heuristic; the Service field on ContainerInfo is preferred.
func extractServiceFromName(containerName string) string {
    parts := strings.Split(containerName, "-")
    if len(parts) < 3 {
        return containerName
    }
    return parts[len(parts)-2]
}

// GetStackList scans the stacks directory and merges with `docker compose ls` status.
// Returns a map of stack name -> Stack. DEPRECATED: use GetStackListFromContainers.
func GetStackList(stacksDir string, composeLsOutput []byte) map[string]*Stack {
    stacks := make(map[string]*Stack)

    // 1. Scan stacks directory for managed stacks
    entries, err := os.ReadDir(stacksDir)
    if err != nil {
        slog.Warn("scan stacks dir", "err", err, "dir", stacksDir)
        return stacks
    }

    for _, entry := range entries {
        if !entry.IsDir() {
            continue
        }
        name := entry.Name()

        // Skip if no compose file exists
        if !ComposeFileExists(stacksDir, name) {
            continue
        }

        s := &Stack{
            Name:              name,
            Status:            CREATED_FILE,
            IsManagedByDockge: true,
            Path:              filepath.Join(stacksDir, name),
        }

        // Detect compose file name
        for _, fname := range acceptedComposeFileNames {
            if _, err := os.Stat(filepath.Join(s.Path, fname)); err == nil {
                s.ComposeFileName = fname
                break
            }
        }

        // Detect override file name
        for _, fname := range acceptedComposeOverrideFileNames {
            if _, err := os.Stat(filepath.Join(s.Path, fname)); err == nil {
                s.ComposeOverrideFileName = fname
                break
            }
        }

        stacks[name] = s
    }

    // 2. Merge status from `docker compose ls`
    if len(composeLsOutput) == 0 {
        return stacks
    }

    entries2, err := ParseComposeLs(composeLsOutput)
    if err != nil {
        slog.Warn("parse compose ls", "err", err)
        return stacks
    }

    for _, entry := range entries2 {
        s, exists := stacks[entry.Name]
        if !exists {
            // Stack not in our managed directory — external stack
            if entry.Name == "dockge" {
                continue // skip the dockge stack itself
            }
            s = &Stack{
                Name:              entry.Name,
                IsManagedByDockge: false,
            }
            stacks[entry.Name] = s
        }
        s.Status = StatusConvert(entry.Status)
    }

    return stacks
}

// BuildStackListJSON converts a stack map to the JSON format the frontend expects.
// updateMap: stack name -> true if any service has an image update available.
// recreateMap: stack name -> true if any running service image differs from compose.yaml.
func BuildStackListJSON(stacks map[string]*Stack, endpoint string, updateMap, recreateMap map[string]bool) map[string]StackSimpleJSON {
    result := make(map[string]StackSimpleJSON, len(stacks))
    for name, s := range stacks {
        result[name] = s.ToSimpleJSON(endpoint, updateMap[name], recreateMap[name])
    }
    return result
}
