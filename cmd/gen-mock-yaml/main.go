// gen-mock-yaml generates mock.yaml files for filler stacks (stack-010 through stack-199)
// and the mega-stack (09-mega-stack). It reads each stack's compose.yaml and writes a
// mock.yaml with deterministic status, command, restart policy, and per-service network
// endpoint config (IP/MAC).
//
// Usage: go run ./cmd/gen-mock-yaml [stacks-dir]
// Default stacks-dir: test-data/stacks
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	stacksDir := "test-data/stacks"
	if len(os.Args) > 1 {
		stacksDir = os.Args[1]
	}

	entries, err := os.ReadDir(stacksDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read stacks dir: %v\n", err)
		os.Exit(1)
	}

	generated := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()

		// Only process filler stacks (stack-NNN) and 09-mega-stack
		isFiller := strings.HasPrefix(name, "stack-")
		isMega := name == "09-mega-stack"
		if !isFiller && !isMega {
			continue
		}

		stackDir := filepath.Join(stacksDir, name)
		composeFile := findComposeFile(stackDir)
		if composeFile == "" {
			continue
		}

		ci := parseCompose(composeFile)
		if len(ci.services) == 0 {
			continue
		}

		var status string
		if isMega {
			status = "running"
		} else {
			status = fillerStatus(name)
		}

		mockPath := filepath.Join(stackDir, "mock.yaml")

		// Collect unique network full names for this stack
		netNames := collectNetworkNames(name, ci)

		// For mega-stack, preserve existing service overrides
		if isMega {
			generateMegaStack(mockPath, name, ci.services, netNames)
		} else {
			generateFillerStack(mockPath, name, status, ci.services, netNames)
		}
		generated++
	}

	fmt.Printf("Generated %d mock.yaml files\n", generated)
}

type serviceInfo struct {
	name     string
	image    string
	networks []string // network names from compose
}

func fillerStatus(name string) string {
	// Extract number from "stack-NNN"
	numStr := strings.TrimPrefix(name, "stack-")
	num := 0
	for _, c := range numStr {
		if c >= '0' && c <= '9' {
			num = num*10 + int(c-'0')
		}
	}
	switch num % 5 {
	case 0, 1, 2:
		return "running"
	case 3:
		return "exited"
	default:
		return "inactive"
	}
}

// collectNetworkNames returns the full Docker network names for a stack,
// sorted deterministically. If the compose file declares explicit networks,
// those are prefixed with stackName_; otherwise stackName_default is used.
func collectNetworkNames(stackName string, ci composeInfo) []string {
	if len(ci.networks) > 0 {
		names := make([]string, len(ci.networks))
		for i, n := range ci.networks {
			names[i] = stackName + "_" + n
		}
		sort.Strings(names)
		return names
	}
	return []string{stackName + "_default"}
}

func generateFillerStack(path, stackName, status string, services []serviceInfo, netNames []string) {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("status: %s\n", status))

	// Write networks section with IDs
	b.WriteString("networks:\n")
	for _, netName := range netNames {
		b.WriteString(fmt.Sprintf("  %s:\n", netName))
		b.WriteString(fmt.Sprintf("    id: \"%s\"\n", mockNetID(netName)))
	}

	if len(services) > 0 {
		b.WriteString("services:\n")

		// Assign IPs starting from a deterministic subnet
		subnetBase := 33 + int(simpleHash(stackName)%150) // 172.33-182.x.x
		ipCounter := 2

		for _, svc := range services {
			b.WriteString(fmt.Sprintf("  %s:\n", svc.name))

			cmd := commandForImage(svc.image)
			if cmd != "" {
				b.WriteString(fmt.Sprintf("    command: \"%s\"\n", cmd))
			}
			b.WriteString("    restart_policy: unless-stopped\n")

			// Determine network name
			netName := stackName + "_default"
			if len(svc.networks) > 0 {
				netName = stackName + "_" + svc.networks[0]
			}

			ip := fmt.Sprintf("172.%d.0.%d", subnetBase, ipCounter)
			mac := fmt.Sprintf("02:42:ac:%02x:00:%02x", subnetBase%256, ipCounter)

			b.WriteString("    networks:\n")
			b.WriteString(fmt.Sprintf("      %s:\n", netName))
			b.WriteString(fmt.Sprintf("        ip: \"%s\"\n", ip))
			b.WriteString(fmt.Sprintf("        mac: \"%s\"\n", mac))

			ipCounter++
		}
	}

	os.WriteFile(path, []byte(b.String()), 0644)
}

func generateMegaStack(path, stackName string, services []serviceInfo, netNames []string) {
	// For mega-stack, preserve existing service state overrides and add network config
	existingOverrides := map[string]string{
		"svc-003": "    state: exited",
		"svc-012": "    state: exited",
		"svc-025": "    state: exited",
		"svc-041": "    health: unhealthy",
		"svc-067": "    health: unhealthy",
		"svc-088": "    state: exited",
	}

	var b strings.Builder
	b.WriteString("status: running\n")

	// Write networks section with IDs
	b.WriteString("networks:\n")
	for _, netName := range netNames {
		b.WriteString(fmt.Sprintf("  %s:\n", netName))
		b.WriteString(fmt.Sprintf("    id: \"%s\"\n", mockNetID(netName)))
	}

	b.WriteString("services:\n")

	subnetBase := 28 // 172.28.x.x for mega-stack
	ipCounter := 2

	for _, svc := range services {
		b.WriteString(fmt.Sprintf("  %s:\n", svc.name))

		if override, ok := existingOverrides[svc.name]; ok {
			b.WriteString(override + "\n")
		}

		b.WriteString("    command: \"sleep infinity\"\n")
		b.WriteString("    restart_policy: unless-stopped\n")

		netName := stackName + "_default"
		ip := fmt.Sprintf("172.%d.0.%d", subnetBase, ipCounter)
		mac := fmt.Sprintf("02:42:ac:%02x:00:%02x", subnetBase%256, ipCounter)

		b.WriteString("    networks:\n")
		b.WriteString(fmt.Sprintf("      %s:\n", netName))
		b.WriteString(fmt.Sprintf("        ip: \"%s\"\n", ip))
		b.WriteString(fmt.Sprintf("        mac: \"%s\"\n", mac))

		ipCounter++
	}

	os.WriteFile(path, []byte(b.String()), 0644)
}

func commandForImage(imageRef string) string {
	name := imageRef
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	if idx := strings.Index(name, ":"); idx >= 0 {
		name = name[:idx]
	}
	switch name {
	case "nginx", "httpd":
		return "nginx -g 'daemon off;'"
	case "redis":
		return "redis-server"
	case "postgres":
		return "postgres"
	case "mysql", "mariadb":
		return "mysqld"
	case "node":
		return "node server.js"
	case "python":
		return "python main.py"
	case "grafana":
		return "/run.sh"
	case "wordpress":
		return "apache2-foreground"
	case "elasticsearch":
		return "/usr/local/bin/docker-entrypoint.sh"
	case "rabbitmq":
		return "rabbitmq-server"
	case "traefik":
		return "/entrypoint.sh traefik"
	case "alpine", "busybox":
		return "sleep infinity"
	case "mongo":
		return "mongod"
	case "memcached":
		return "memcached"
	case "minio":
		return "minio server /data"
	default:
		return "/docker-entrypoint.sh"
	}
}

func simpleHash(s string) uint64 {
	var h uint64 = 5381
	for _, c := range s {
		h = h*33 + uint64(c)
	}
	return h
}

// mockHash generates a deterministic 32-char hex hash from a string.
// Identical to fakedaemon.go's mockHash — must stay in sync.
func mockHash(s string) string {
	var h uint64 = 14695981039346656037 // FNV offset basis
	for _, c := range s {
		h ^= uint64(c)
		h *= 1099511628211 // FNV prime
	}
	return fmt.Sprintf("%016x%016x", h, h^0xdeadbeefcafebabe)
}

// mockNetID returns a 64-char hex network ID (matching real Docker network IDs).
func mockNetID(name string) string {
	h := mockHash(name)
	return h + h
}

// composeInfo holds data extracted from a compose.yaml file.
type composeInfo struct {
	services []serviceInfo
	networks []string // top-level network names
}

// parseCompose extracts service names, images, and top-level networks from a compose.yaml.
func parseCompose(path string) composeInfo {
	f, err := os.Open(path)
	if err != nil {
		return composeInfo{}
	}
	defer f.Close()

	var ci composeInfo
	var current *serviceInfo
	section := ""

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimRight(line, " \t")
		if trimmed == "" || strings.HasPrefix(strings.TrimSpace(trimmed), "#") {
			continue
		}

		indent := countIndent(line)

		if indent == 0 {
			stripped := strings.TrimSuffix(strings.TrimSpace(trimmed), ":")
			switch stripped {
			case "services":
				section = "services"
			case "networks":
				section = "networks"
			default:
				section = ""
			}
			current = nil
			continue
		}

		switch section {
		case "services":
			if indent == 2 && strings.HasSuffix(trimmed, ":") {
				name := strings.TrimSpace(strings.TrimSuffix(trimmed, ":"))
				ci.services = append(ci.services, serviceInfo{name: name})
				current = &ci.services[len(ci.services)-1]
				continue
			}
			if current != nil && indent >= 4 {
				field := strings.TrimSpace(trimmed)
				if strings.HasPrefix(field, "image:") {
					img := strings.TrimSpace(strings.TrimPrefix(field, "image:"))
					img = strings.Trim(img, "\"'")
					current.image = img
				}
			}
		case "networks":
			if indent == 2 && strings.HasSuffix(trimmed, ":") {
				name := strings.TrimSpace(strings.TrimSuffix(trimmed, ":"))
				ci.networks = append(ci.networks, name)
			}
		}
	}

	// Sort for deterministic output
	sort.Slice(ci.services, func(i, j int) bool {
		return ci.services[i].name < ci.services[j].name
	})
	sort.Strings(ci.networks)

	return ci
}

// parseServices extracts service names and images from a compose.yaml file.
// Wrapper for backward compatibility.
func parseServices(path string) []serviceInfo {
	return parseCompose(path).services
}

func countIndent(line string) int {
	for i, c := range line {
		if c != ' ' && c != '\t' {
			return i
		}
	}
	return len(line)
}

func findComposeFile(dir string) string {
	for _, name := range []string{"compose.yaml", "docker-compose.yaml", "docker-compose.yml", "compose.yml"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
