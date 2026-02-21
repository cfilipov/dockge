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
	}

	fmt.Println("Generated 190 filler stacks (stack-010 through stack-199)")
}

func generateCompose(stackIdx, count int) string {
	var b strings.Builder
	b.WriteString("services:\n")

	for s := 0; s < count; s++ {
		img := images[(stackIdx*7+s*3)%len(images)]
		svcName := serviceName(img, s, count)

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
	}

	return b.String()
}

// serviceName picks a descriptive service name based on the image.
// For multi-service stacks, appends a suffix to avoid collisions.
func serviceName(img string, svcIdx, total int) string {
	// Extract base name from image (before : and /)
	name := img
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	if idx := strings.Index(name, ":"); idx >= 0 {
		name = name[:idx]
	}

	if total > 1 && svcIdx > 0 {
		return fmt.Sprintf("%s-%d", name, svcIdx)
	}
	return name
}
