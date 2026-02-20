package stack

import (
    "log/slog"
    "os"
    "path/filepath"
)

// GetStackList scans the stacks directory and merges with `docker compose ls` status.
// Returns a map of stack name → Stack.
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
func BuildStackListJSON(stacks map[string]*Stack, endpoint string) map[string]interface{} {
    result := make(map[string]interface{}, len(stacks))
    for name, s := range stacks {
        result[name] = s.ToSimpleJSON(endpoint)
    }
    return result
}
