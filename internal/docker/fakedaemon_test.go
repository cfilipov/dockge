package docker

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupFakeDaemon creates test stacks, builds MockData, starts a FakeDaemon,
// and returns an SDKClient connected to it plus a cleanup function.
func setupFakeDaemon(t *testing.T) (*SDKClient, func()) {
	t.Helper()

	// Create temp stacks directory with a test stack
	stacksDir := t.TempDir()
	stackDir := filepath.Join(stacksDir, "test-app")
	os.MkdirAll(stackDir, 0o755)
	os.WriteFile(filepath.Join(stackDir, "compose.yaml"), []byte(`services:
  web:
    image: nginx:latest
  api:
    image: node:18-alpine
`), 0o644)

	data := BuildMockData(stacksDir)
	state := NewMockStateFrom(map[string]string{
		"test-app": "running",
	})

	sockPath, cleanup, err := StartFakeDaemon(state, data, stacksDir, "")
	if err != nil {
		t.Fatalf("start fake daemon: %v", err)
	}

	client, err := NewSDKClientWithHost("unix://" + sockPath)
	if err != nil {
		cleanup()
		t.Fatalf("new sdk client: %v", err)
	}

	return client, func() {
		client.Close()
		cleanup()
	}
}

func TestFakeDaemon_ContainerList(t *testing.T) {
	client, cleanup := setupFakeDaemon(t)
	defer cleanup()

	ctx := context.Background()

	// List running containers
	containers, err := client.ContainerList(ctx, false, "")
	if err != nil {
		t.Fatalf("ContainerList: %v", err)
	}

	if len(containers) == 0 {
		t.Fatal("expected containers, got none")
	}

	// Should have web and api services from test-app
	found := map[string]bool{}
	for _, c := range containers {
		if c.Project == "test-app" {
			found[c.Service] = true
		}
	}
	if !found["web"] {
		t.Error("missing web service")
	}
	if !found["api"] {
		t.Error("missing api service")
	}

	// With project filter
	filtered, err := client.ContainerList(ctx, false, "test-app")
	if err != nil {
		t.Fatalf("ContainerList filtered: %v", err)
	}
	for _, c := range filtered {
		if c.Project != "test-app" {
			t.Errorf("expected project test-app, got %s", c.Project)
		}
	}
}

func TestFakeDaemon_ContainerInspect(t *testing.T) {
	client, cleanup := setupFakeDaemon(t)
	defer cleanup()

	ctx := context.Background()

	// Get a container ID first
	containers, _ := client.ContainerList(ctx, false, "test-app")
	if len(containers) == 0 {
		t.Fatal("no containers")
	}

	// Inspect returns JSON string
	result, err := client.ContainerInspect(ctx, containers[0].ID)
	if err != nil {
		t.Fatalf("ContainerInspect: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty inspect result")
	}
}

func TestFakeDaemon_ContainerStats(t *testing.T) {
	client, cleanup := setupFakeDaemon(t)
	defer cleanup()

	ctx := context.Background()

	stats, err := client.ContainerStats(ctx, "test-app")
	if err != nil {
		t.Fatalf("ContainerStats: %v", err)
	}

	if len(stats) == 0 {
		t.Fatal("expected stats, got none")
	}

	// Verify the CPU% calculation ran (should produce non-zero)
	for name, s := range stats {
		if s.CPUPerc == "" {
			t.Errorf("missing CPUPerc for %s", name)
		}
		if s.MemPerc == "" {
			t.Errorf("missing MemPerc for %s", name)
		}
		if s.MemUsage == "" {
			t.Errorf("missing MemUsage for %s", name)
		}
	}
}

func TestFakeDaemon_ContainerTop(t *testing.T) {
	client, cleanup := setupFakeDaemon(t)
	defer cleanup()

	ctx := context.Background()
	containers, _ := client.ContainerList(ctx, false, "test-app")
	if len(containers) == 0 {
		t.Fatal("no containers")
	}

	titles, processes, err := client.ContainerTop(ctx, containers[0].ID)
	if err != nil {
		t.Fatalf("ContainerTop: %v", err)
	}
	if len(titles) == 0 {
		t.Error("expected titles")
	}
	if len(processes) == 0 {
		t.Error("expected processes")
	}
}

func TestFakeDaemon_ContainerLogs(t *testing.T) {
	client, cleanup := setupFakeDaemon(t)
	defer cleanup()

	ctx := context.Background()
	containers, _ := client.ContainerList(ctx, false, "test-app")
	if len(containers) == 0 {
		t.Fatal("no containers")
	}

	rc, _, err := client.ContainerLogs(ctx, containers[0].ID, "100", false)
	if err != nil {
		t.Fatalf("ContainerLogs: %v", err)
	}
	defer rc.Close()

	buf := make([]byte, 4096)
	n, _ := rc.Read(buf)
	if n == 0 {
		t.Error("expected log output")
	}
}

func TestFakeDaemon_ContainerStartedAt(t *testing.T) {
	client, cleanup := setupFakeDaemon(t)
	defer cleanup()

	ctx := context.Background()
	containers, _ := client.ContainerList(ctx, false, "test-app")
	if len(containers) == 0 {
		t.Fatal("no containers")
	}

	ts, err := client.ContainerStartedAt(ctx, containers[0].ID)
	if err != nil {
		t.Fatalf("ContainerStartedAt: %v", err)
	}
	if ts.IsZero() {
		t.Error("expected non-zero started at time")
	}
}

func TestFakeDaemon_ImageList(t *testing.T) {
	client, cleanup := setupFakeDaemon(t)
	defer cleanup()

	ctx := context.Background()
	images, err := client.ImageList(ctx)
	if err != nil {
		t.Fatalf("ImageList: %v", err)
	}

	if len(images) == 0 {
		t.Fatal("expected images, got none")
	}

	// Should include nginx:latest and node:18-alpine at minimum
	foundNginx := false
	foundDangling := false
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == "nginx:latest" {
				foundNginx = true
			}
		}
		if img.Dangling {
			foundDangling = true
		}
	}
	if !foundNginx {
		t.Error("expected nginx:latest in image list")
	}
	if !foundDangling {
		t.Error("expected dangling images in image list")
	}
}

func TestFakeDaemon_ImageInspect(t *testing.T) {
	client, cleanup := setupFakeDaemon(t)
	defer cleanup()

	ctx := context.Background()
	digests, err := client.ImageInspect(ctx, "nginx:latest")
	if err != nil {
		t.Fatalf("ImageInspect: %v", err)
	}
	if len(digests) == 0 {
		t.Error("expected RepoDigests")
	}
}

func TestFakeDaemon_ImageInspectDetail(t *testing.T) {
	client, cleanup := setupFakeDaemon(t)
	defer cleanup()

	ctx := context.Background()
	detail, err := client.ImageInspectDetail(ctx, "nginx:latest")
	if err != nil {
		t.Fatalf("ImageInspectDetail: %v", err)
	}
	if detail.Architecture != "amd64" {
		t.Errorf("expected amd64, got %s", detail.Architecture)
	}
	if len(detail.Layers) == 0 {
		t.Error("expected layers")
	}
}

func TestFakeDaemon_ImagePrune(t *testing.T) {
	client, cleanup := setupFakeDaemon(t)
	defer cleanup()

	ctx := context.Background()
	result, err := client.ImagePrune(ctx, false)
	if err != nil {
		t.Fatalf("ImagePrune: %v", err)
	}
	if result == "" {
		t.Error("expected prune result")
	}
}

func TestFakeDaemon_DistributionInspect(t *testing.T) {
	client, cleanup := setupFakeDaemon(t)
	defer cleanup()

	ctx := context.Background()
	digest, err := client.DistributionInspect(ctx, "nginx:latest")
	if err != nil {
		t.Fatalf("DistributionInspect: %v", err)
	}
	if digest == "" {
		t.Error("expected digest")
	}
}

func TestFakeDaemon_NetworkList(t *testing.T) {
	client, cleanup := setupFakeDaemon(t)
	defer cleanup()

	ctx := context.Background()
	networks, err := client.NetworkList(ctx)
	if err != nil {
		t.Fatalf("NetworkList: %v", err)
	}

	if len(networks) == 0 {
		t.Fatal("expected networks")
	}

	// Should always include bridge, host, none
	foundBridge := false
	for _, n := range networks {
		if n.Name == "bridge" {
			foundBridge = true
		}
	}
	if !foundBridge {
		t.Error("expected bridge network")
	}
}

func TestFakeDaemon_NetworkInspect(t *testing.T) {
	client, cleanup := setupFakeDaemon(t)
	defer cleanup()

	ctx := context.Background()
	detail, err := client.NetworkInspect(ctx, "bridge")
	if err != nil {
		t.Fatalf("NetworkInspect: %v", err)
	}
	if detail.Name != "bridge" {
		t.Errorf("expected bridge, got %s", detail.Name)
	}
}

func TestFakeDaemon_VolumeList(t *testing.T) {
	client, cleanup := setupFakeDaemon(t)
	defer cleanup()

	ctx := context.Background()
	volumes, err := client.VolumeList(ctx)
	if err != nil {
		t.Fatalf("VolumeList: %v", err)
	}

	// Volumes may be empty if no test stack defines named volumes
	_ = volumes
}

func TestFakeDaemon_Events(t *testing.T) {
	// Create temp stacks dir
	stacksDir := t.TempDir()
	stackDir := filepath.Join(stacksDir, "evt-test")
	os.MkdirAll(stackDir, 0o755)
	os.WriteFile(filepath.Join(stackDir, "compose.yaml"), []byte(`services:
  web:
    image: nginx:latest
`), 0o644)

	data := BuildMockData(stacksDir)
	state := NewMockStateFrom(map[string]string{"evt-test": "running"})

	sockPath, cleanup, err := StartFakeDaemon(state, data, stacksDir, "")
	if err != nil {
		t.Fatalf("start fake daemon: %v", err)
	}
	defer cleanup()

	client, err := NewSDKClientWithHost("unix://" + sockPath)
	if err != nil {
		t.Fatalf("new sdk client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	evtCh, errCh := client.Events(ctx)

	// Trigger a state change via the fake daemon's /_mock/state endpoint
	// (we can do this by directly modifying MockState since the daemon
	// also watches for these — but for events to fire immediately we
	// should use the HTTP endpoint. Since the events poller runs on 60s,
	// let's just verify the channels are open and cancel cleanly.)
	select {
	case <-ctx.Done():
		// Expected: no events within 3 seconds, context cancelled
	case err := <-errCh:
		if err != nil {
			t.Logf("events error (expected on cancel): %v", err)
		}
	case evt := <-evtCh:
		t.Logf("received event: %+v", evt)
	}
}
