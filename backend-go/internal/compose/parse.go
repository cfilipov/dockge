package compose

import (
    "bufio"
    "os"
    "strings"
)

// ServiceData holds the extracted per-service data from a compose file.
type ServiceData struct {
    Image             string // e.g. "nginx:latest"
    StatusIgnore      bool   // dockge.status.ignore == "true"
    ImageUpdatesCheck bool   // dockge.imageupdates.check != "false" (default: true)
}

// ParseFile reads a compose file from disk and extracts service data.
func ParseFile(path string) map[string]ServiceData {
    f, err := os.Open(path)
    if err != nil {
        return nil
    }
    defer f.Close()
    return parseScanner(bufio.NewScanner(f))
}

// ParseYAML parses compose YAML from a string and extracts service data.
func ParseYAML(yaml string) map[string]ServiceData {
    return parseScanner(bufio.NewScanner(strings.NewReader(yaml)))
}

// parseScanner is the shared line-by-line parser that extracts service data.
// It recognizes:
//   - Service names (2-space indent under "services:", ends with ":")
//   - image: values (4+ space indent under a service)
//   - labels: block (4-space indent under a service)
//   - dockge.* label key-value pairs (6+ space indent under labels)
func parseScanner(scanner *bufio.Scanner) map[string]ServiceData {
    result := make(map[string]ServiceData)

    inServices := false
    currentService := ""
    inLabels := false

    for scanner.Scan() {
        line := scanner.Text()
        trimmed := strings.TrimRight(line, " \t")

        // Detect "services:" top-level key
        if trimmed == "services:" {
            inServices = true
            currentService = ""
            inLabels = false
            continue
        }
        if !inServices {
            continue
        }

        // Exit services block on next top-level key (non-indented, non-comment, non-empty)
        if len(trimmed) > 0 && trimmed[0] != ' ' && trimmed[0] != '#' {
            break
        }

        // Empty or comment-only line — preserve context
        if len(trimmed) == 0 || strings.TrimSpace(trimmed) == "" {
            continue
        }
        if strings.TrimSpace(trimmed)[0] == '#' {
            continue
        }

        // Service name: exactly 2-space indent, non-space third char, ends with ":"
        if len(line) > 2 && line[0] == ' ' && line[1] == ' ' && line[2] != ' ' && strings.HasSuffix(trimmed, ":") {
            currentService = strings.TrimSpace(strings.TrimSuffix(trimmed, ":"))
            inLabels = false
            // Initialize with default: ImageUpdatesCheck = true
            result[currentService] = ServiceData{ImageUpdatesCheck: true}
            continue
        }

        if currentService == "" {
            continue
        }

        // Count leading spaces
        indent := 0
        for _, ch := range line {
            if ch == ' ' {
                indent++
            } else {
                break
            }
        }

        // 4-space indent: service-level keys
        if indent >= 4 && indent < 6 {
            stripped := strings.TrimSpace(trimmed)

            // image: field
            if strings.HasPrefix(stripped, "image:") {
                img := strings.TrimSpace(strings.TrimPrefix(stripped, "image:"))
                if img != "" {
                    sd := result[currentService]
                    sd.Image = img
                    result[currentService] = sd
                }
                inLabels = false
                continue
            }

            // labels: block
            if stripped == "labels:" {
                inLabels = true
                continue
            }

            // Any other 4-space key exits labels context
            inLabels = false
            continue
        }

        // 6+ space indent: label entries (only if we're inside a labels block)
        if indent >= 6 && inLabels {
            stripped := strings.TrimSpace(trimmed)

            // Only care about dockge.* labels
            if !strings.HasPrefix(stripped, "dockge.") {
                continue
            }

            // Parse key: value (handle both "key: value" and "key: \"value\"")
            colonIdx := strings.Index(stripped, ":")
            if colonIdx < 0 {
                continue
            }
            key := stripped[:colonIdx]
            val := strings.TrimSpace(stripped[colonIdx+1:])
            // Remove surrounding quotes
            val = strings.Trim(val, "\"'")

            sd := result[currentService]
            switch key {
            case "dockge.status.ignore":
                sd.StatusIgnore = val == "true"
            case "dockge.imageupdates.check":
                sd.ImageUpdatesCheck = val != "false"
            }
            result[currentService] = sd
        }
    }

    return result
}
