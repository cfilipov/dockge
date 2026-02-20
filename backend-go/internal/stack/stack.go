package stack

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

// Status constants — must match common/util-common.ts
const (
    UNKNOWN            = 0
    CREATED_FILE       = 1
    CREATED_STACK      = 2
    RUNNING            = 3
    EXITED             = 4
    RUNNING_AND_EXITED = 5
    UNHEALTHY          = 6
)

// Accepted compose file names (checked in order)
var acceptedComposeFileNames = []string{
    "compose.yaml",
    "docker-compose.yaml",
    "docker-compose.yml",
    "compose.yml",
}

var acceptedComposeOverrideFileNames = []string{
    "compose.override.yaml",
    "compose.override.yml",
    "docker-compose.override.yaml",
    "docker-compose.override.yml",
}

// Stack represents a docker compose stack.
type Stack struct {
    Name                    string
    Status                  int
    IsManagedByDockge       bool
    ComposeFileName         string
    ComposeOverrideFileName string
    ComposeYAML             string
    ComposeENV              string
    ComposeOverrideYAML     string
    Path                    string // full path to stack directory
}

// IsStarted returns true if the stack has running containers.
func (s *Stack) IsStarted() bool {
    return s.Status == RUNNING || s.Status == RUNNING_AND_EXITED || s.Status == UNHEALTHY
}

// ToSimpleJSON returns the stack data for the stack list broadcast.
// stackUpdates indicates whether this stack has image updates available.
// stackRecreate indicates whether this stack has containers needing recreation.
func (s *Stack) ToSimpleJSON(endpoint string, hasUpdates, recreateNecessary bool) map[string]interface{} {
    return map[string]interface{}{
        "name":                    s.Name,
        "status":                  s.Status,
        "started":                 s.IsStarted(),
        "recreateNecessary":       recreateNecessary,
        "tags":                    []string{},
        "isManagedByDockge":       s.IsManagedByDockge,
        "composeFileName":         s.ComposeFileName,
        "composeOverrideFileName": s.ComposeOverrideFileName,
        "endpoint":                endpoint,
        "imageUpdatesAvailable":   hasUpdates,
    }
}

// ToJSON returns full stack data including YAML content (for getStack).
func (s *Stack) ToJSON(endpoint, primaryHostname string, hasUpdates, recreateNecessary bool) map[string]interface{} {
    obj := s.ToSimpleJSON(endpoint, hasUpdates, recreateNecessary)
    obj["composeYAML"] = s.ComposeYAML
    obj["composeENV"] = s.ComposeENV
    obj["composeOverrideYAML"] = s.ComposeOverrideYAML
    obj["primaryHostname"] = primaryHostname
    return obj
}

// LoadFromDisk reads the compose files from the stack directory.
func (s *Stack) LoadFromDisk(stacksDir string) error {
    s.Path = filepath.Join(stacksDir, s.Name)

    // Find compose file
    for _, name := range acceptedComposeFileNames {
        path := filepath.Join(s.Path, name)
        if data, err := os.ReadFile(path); err == nil {
            s.ComposeFileName = name
            s.ComposeYAML = string(data)
            break
        }
    }

    // Find override file
    for _, name := range acceptedComposeOverrideFileNames {
        path := filepath.Join(s.Path, name)
        if data, err := os.ReadFile(path); err == nil {
            s.ComposeOverrideFileName = name
            s.ComposeOverrideYAML = string(data)
            break
        }
    }

    // Read .env file
    envPath := filepath.Join(s.Path, ".env")
    if data, err := os.ReadFile(envPath); err == nil {
        s.ComposeENV = string(data)
    }

    return nil
}

// SaveToDisk writes the compose files to the stack directory.
func (s *Stack) SaveToDisk(stacksDir string) error {
    s.Path = filepath.Join(stacksDir, s.Name)

    // Create directory
    if err := os.MkdirAll(s.Path, 0755); err != nil {
        return fmt.Errorf("create stack dir: %w", err)
    }

    // Determine compose file name
    composeFile := s.ComposeFileName
    if composeFile == "" {
        composeFile = "compose.yaml"
        s.ComposeFileName = composeFile
    }

    // Write compose file
    if err := os.WriteFile(filepath.Join(s.Path, composeFile), []byte(s.ComposeYAML), 0644); err != nil {
        return fmt.Errorf("write compose file: %w", err)
    }

    // Write .env if non-empty
    envPath := filepath.Join(s.Path, ".env")
    if s.ComposeENV != "" {
        if err := os.WriteFile(envPath, []byte(s.ComposeENV), 0644); err != nil {
            return fmt.Errorf("write env file: %w", err)
        }
    } else {
        os.Remove(envPath) // clean up if empty
    }

    // Write override file if non-empty
    if s.ComposeOverrideYAML != "" {
        overrideFile := s.ComposeOverrideFileName
        if overrideFile == "" {
            overrideFile = "compose.override.yaml"
            s.ComposeOverrideFileName = overrideFile
        }
        if err := os.WriteFile(filepath.Join(s.Path, overrideFile), []byte(s.ComposeOverrideYAML), 0644); err != nil {
            return fmt.Errorf("write override file: %w", err)
        }
    }

    return nil
}

// ComposeFileExists checks if any accepted compose file exists for a stack.
func ComposeFileExists(stacksDir, stackName string) bool {
    dir := filepath.Join(stacksDir, stackName)
    for _, name := range acceptedComposeFileNames {
        if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
            return true
        }
    }
    return false
}

// StatusConvert converts the status string from `docker compose ls` to a status number.
// Input examples: "running(2)", "exited(2)", "running(2), exited(1)", "created(1)"
func StatusConvert(statusStr string) int {
    if strings.HasPrefix(statusStr, "created") {
        return CREATED_STACK
    }

    runningCount := parseStatusCount(statusStr, "running")
    exitedCount := parseStatusCount(statusStr, "exited")

    if runningCount > 0 && exitedCount > 0 {
        return RUNNING_AND_EXITED
    } else if runningCount > 0 {
        return RUNNING
    } else if exitedCount > 0 {
        return EXITED
    } else if strings.Contains(statusStr, "running") {
        return RUNNING
    } else if strings.Contains(statusStr, "exited") {
        return EXITED
    }

    return UNKNOWN
}

func parseStatusCount(status, keyword string) int {
    idx := strings.Index(status, keyword+"(")
    if idx < 0 {
        return 0
    }
    rest := status[idx+len(keyword)+1:]
    end := strings.Index(rest, ")")
    if end < 0 {
        return 0
    }
    var n int
    fmt.Sscanf(rest[:end], "%d", &n)
    return n
}

// ComposeLsEntry is one entry from `docker compose ls --format json`.
type ComposeLsEntry struct {
    Name        string `json:"Name"`
    Status      string `json:"Status"`
    ConfigFiles string `json:"ConfigFiles"`
}

// ParseComposeLs parses the JSON output of `docker compose ls --format json`.
func ParseComposeLs(data []byte) ([]ComposeLsEntry, error) {
    var entries []ComposeLsEntry
    if err := json.Unmarshal(data, &entries); err != nil {
        return nil, err
    }
    return entries, nil
}
