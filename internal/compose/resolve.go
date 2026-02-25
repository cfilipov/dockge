package compose

import (
    "os"
    "path/filepath"
)

// Accepted compose file names (checked in order).
// Duplicated from stack package to avoid circular import.
var acceptedComposeFileNames = []string{
    "compose.yaml",
    "docker-compose.yaml",
    "docker-compose.yml",
    "compose.yml",
}

// FindComposeFile returns the full path to the compose file for a stack,
// checking accepted file names in order. Returns empty string if none found.
func FindComposeFile(stacksDir, stackName string) string {
    dir := filepath.Join(stacksDir, stackName)
    for _, name := range acceptedComposeFileNames {
        path := filepath.Join(dir, name)
        if _, err := os.Stat(path); err == nil {
            return path
        }
    }
    return ""
}
