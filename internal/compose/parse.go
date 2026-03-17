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

// parseScanner is a fast, line-by-line YAML parser that extracts service data
// without a full YAML parse. It recognizes:
//   - Service names (2-space indent under "services:", ends with ":")
//   - image: values (4+ space indent under a service)
//   - labels: block (4-space indent under a service)
//   - dockge.* label key-value pairs (6+ space indent under labels)
//
// Assumptions and limitations:
//   - Indentation uses spaces only (no tabs). Standard for Docker Compose.
//   - Service names are at exactly 2-space indent under "services:".
//   - Service-level keys (image, labels) are at 4-space indent.
//   - Label entries are at 6+ space indent.
//   - Inline YAML comments (# after whitespace) are stripped from values.
//   - Only the first "services:" block is parsed; subsequent ones are ignored.
//   - Anchors, aliases, and flow mappings ({}) are not supported.
//
// These trade-offs are intentional: full YAML parsing (via gopkg.in/yaml.v3)
// is ~100x slower and allocates heavily. Since compose files follow a strict
// subset of YAML, this line scanner handles real-world files correctly.
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
                img := stripInlineComment(strings.TrimSpace(strings.TrimPrefix(stripped, "image:")))
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
            val := stripInlineComment(strings.TrimSpace(stripped[colonIdx+1:]))
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

// stripInlineComment removes a YAML inline comment from a value.
// In YAML, a comment starts with " #" (space + hash). A bare "#" without
// a preceding space is NOT a comment — it's common in image tags like
// "myregistry.io/image#sha256:abc". Quoted values are returned as-is
// since the comment would be inside the quotes.
func stripInlineComment(s string) string {
    if s == "" {
        return s
    }
    // If the value is quoted, don't strip — the # is part of the value
    if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
        return s
    }
    if idx := strings.Index(s, " #"); idx >= 0 {
        return strings.TrimRight(s[:idx], " \t")
    }
    return s
}
