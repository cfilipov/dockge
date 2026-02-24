// gen-teststacks deterministically generates 190 filler test stacks
// (stack-010 through stack-199) under backend-go/test-data/stacks/.
//
// Usage:
//
//	cd backend-go && go run ./cmd/gen-teststacks
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var images = []string{
	"nginx:latest",
	"redis:7",
	"postgres:16",
	"mysql:8.0",
	"mongo:7",
	"node:20-alpine",
	"python:3.12-slim",
	"httpd:2.4",
	"memcached:1.6",
	"rabbitmq:3-management",
	"mariadb:11",
	"grafana/grafana:latest",
	"elasticsearch:8.12.0",
	"traefik:v3.0",
	"caddy:2",
	"busybox:latest",
	"alpine:latest",
	"ubuntu:24.04",
	"debian:bookworm-slim",
	"golang:1.22",
	"ruby:3.3-slim",
	"php:8.3-fpm",
	"hashicorp/vault:1.15",
	"hashicorp/consul:1.17",
	"minio/minio:latest",
}

// dbImages are images that typically use persistent volumes.
var dbImages = map[string]string{
	"postgres":      "/var/lib/postgresql/data",
	"mysql":         "/var/lib/mysql",
	"mariadb":       "/var/lib/mysql",
	"mongo":         "/data/db",
	"redis":         "/data",
	"elasticsearch": "/usr/share/elasticsearch/data",
	"minio":         "/data",
}

// serviceCount returns the number of services for a given stack index (10–199).
// Distribution: ~100 single, ~40 double, ~25 triple, ~15 quad/quint, ~8 mid, ~2 large.
func serviceCount(i int) int {
	mod := i % 10
	switch {
	case mod < 6:
		return 1
	case mod < 8:
		return 2
	case mod == 8:
		r := (i / 10) % 5
		switch {
		case r < 3:
			return 3
		case r < 4:
			return 4
		default:
			return 5
		}
	default: // mod == 9
		r := (i / 10) % 4
		switch {
		case r < 2:
			return 6 + (i/20)%5
		case r < 3:
			return 15
		default:
			return 20
		}
	}
}

func main() {
	outDir := filepath.Join("test-data", "stacks")

	mockCount := 0
	for i := 10; i < 200; i++ {
		name := fmt.Sprintf("stack-%03d", i)
		dir := filepath.Join(outDir, name)

		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "mkdir %s: %v\n", dir, err)
			os.Exit(1)
		}

		count := serviceCount(i)
		yaml := generateCompose(i, count)

		path := filepath.Join(dir, "compose.yaml")
		if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "write %s: %v\n", path, err)
			os.Exit(1)
		}

		// Generate mock.yaml for some multi-service stacks
		if mockYAML := generateMockYAML(i, count); mockYAML != "" {
			mockPath := filepath.Join(dir, "mock.yaml")
			if err := os.WriteFile(mockPath, []byte(mockYAML), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "write %s: %v\n", mockPath, err)
				os.Exit(1)
			}
			mockCount++
		}
	}

	fmt.Printf("Generated 190 filler stacks (stack-010 through stack-199), %d with mock.yaml\n", mockCount)
}

type serviceInfo struct {
	name  string
	image string
}

func generateCompose(stackIdx, count int) string {
	var b strings.Builder
	b.WriteString("services:\n")

	svcs := make([]serviceInfo, 0, count)
	var namedVolumes []string
	hasNetworks := count >= 3

	for s := 0; s < count; s++ {
		img := images[(stackIdx*7+s*3)%len(images)]
		svcName := serviceName(img, s, count)
		svcs = append(svcs, serviceInfo{name: svcName, image: img})

		b.WriteString(fmt.Sprintf("  %s:\n", svcName))
		b.WriteString(fmt.Sprintf("    image: %s\n", img))

		// Add a command for images that need one to stay running
		switch {
		case strings.HasPrefix(img, "busybox:"),
			strings.HasPrefix(img, "alpine:"),
			strings.HasPrefix(img, "ubuntu:"),
			strings.HasPrefix(img, "debian:"),
			strings.HasPrefix(img, "golang:"),
			strings.HasPrefix(img, "ruby:"),
			strings.HasPrefix(img, "php:"),
			strings.HasPrefix(img, "node:"),
			strings.HasPrefix(img, "python:"):
			b.WriteString("    command: sleep infinity\n")
		}

		b.WriteString("    restart: unless-stopped\n")

		// Add a port mapping for the first service
		if s == 0 {
			port := 10000 + stackIdx
			b.WriteString(fmt.Sprintf("    ports:\n      - \"%d:80\"\n", port))
		}

		// Add volume for database images in multi-service stacks
		if count > 1 {
			baseName := imageBaseName(img)
			if mountPath, ok := dbImages[baseName]; ok {
				volName := svcName + "-data"
				namedVolumes = append(namedVolumes, volName)
				b.WriteString(fmt.Sprintf("    volumes:\n      - %s:%s\n", volName, mountPath))
			}
		}

		// Add network membership for stacks with 3+ services
		if hasNetworks {
			b.WriteString("    networks:\n      - app-net\n")
		}
	}

	// Top-level volumes
	if len(namedVolumes) > 0 {
		b.WriteString("\nvolumes:\n")
		for _, v := range namedVolumes {
			b.WriteString(fmt.Sprintf("  %s:\n", v))
		}
	}

	// Top-level networks
	if hasNetworks {
		b.WriteString("\nnetworks:\n  app-net:\n")
	}

	return b.String()
}

// generateMockYAML creates a mock.yaml for some filler stacks to add variety.
// ~10% of multi-service stacks get one service marked exited.
// ~5% get an update_available flag.
func generateMockYAML(stackIdx, count int) string {
	if count < 2 {
		return ""
	}

	var b strings.Builder

	// ~10% of multi-service stacks: one service exited
	if stackIdx%20 == 8 && count >= 2 {
		// Pick the second service
		img := images[(stackIdx*7+1*3)%len(images)]
		svcName := serviceName(img, 1, count)
		b.WriteString("status: running\n")
		b.WriteString("services:\n")
		b.WriteString(fmt.Sprintf("  %s:\n", svcName))
		b.WriteString("    state: exited\n")
		return b.String()
	}

	// ~5% of multi-service stacks: update available on first service
	if stackIdx%40 == 16 && count >= 2 {
		img := images[(stackIdx*7)%len(images)]
		svcName := serviceName(img, 0, count)
		b.WriteString("status: running\n")
		b.WriteString("services:\n")
		b.WriteString(fmt.Sprintf("  %s:\n", svcName))
		b.WriteString("    update_available: true\n")
		return b.String()
	}

	return ""
}

// serviceName picks a descriptive service name based on the image.
// For multi-service stacks, appends a suffix to avoid collisions.
func serviceName(img string, svcIdx, total int) string {
	name := imageBaseName(img)
	if total > 1 && svcIdx > 0 {
		return fmt.Sprintf("%s-%d", name, svcIdx)
	}
	return name
}

// imageBaseName extracts the base name from an image reference.
// "grafana/grafana:latest" → "grafana", "postgres:16" → "postgres"
func imageBaseName(img string) string {
	name := img
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	if idx := strings.Index(name, ":"); idx >= 0 {
		name = name[:idx]
	}
	return name
}
