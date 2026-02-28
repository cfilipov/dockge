package docker

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestWorld(t *testing.T) (*MockWorld, string) {
	t.Helper()

	stacksDir := t.TempDir()

	// Create a stack with explicit network config in mock.yaml
	stackDir := filepath.Join(stacksDir, "test-app")
	os.MkdirAll(stackDir, 0o755)
	os.WriteFile(filepath.Join(stackDir, "compose.yaml"), []byte(`services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
    networks:
      - frontend
  db:
    image: postgres:16
    volumes:
      - pgdata:/var/lib/postgresql/data
    networks:
      - backend

volumes:
  pgdata:

networks:
  frontend:
  backend:
`), 0o644)
	os.WriteFile(filepath.Join(stackDir, "mock.yaml"), []byte(`status: running
services:
  web:
    command: "nginx -g 'daemon off;'"
    restart_policy: unless-stopped
    networks:
      test-app_frontend:
        ip: "172.20.0.2"
        mac: "02:42:ac:14:00:02"
  db:
    health: healthy
    command: "postgres"
    restart_policy: unless-stopped
    networks:
      test-app_backend:
        ip: "172.21.0.2"
        mac: "02:42:ac:15:00:02"
`), 0o644)

	data := BuildMockData(stacksDir)
	state := NewMockStateFrom(map[string]string{
		"test-app": "running",
	})

	world := BuildMockWorld(data, state, stacksDir)
	return world, stacksDir
}

func TestMockWorld_ContainerList(t *testing.T) {
	world, _ := setupTestWorld(t)

	containers := world.ContainerList(true, "")
	if len(containers) == 0 {
		t.Fatal("expected containers")
	}

	// Find test-app containers
	found := map[string]bool{}
	for _, c := range containers {
		if c.Labels["com.docker.compose.project"] == "test-app" {
			found[c.Labels["com.docker.compose.service"]] = true
		}
	}
	if !found["web"] {
		t.Error("missing web service")
	}
	if !found["db"] {
		t.Error("missing db service")
	}
}

func TestMockWorld_ContainerList_ProjectFilter(t *testing.T) {
	world, _ := setupTestWorld(t)

	filtered := world.ContainerList(true, "test-app")
	for _, c := range filtered {
		if c.Labels["com.docker.compose.project"] != "test-app" {
			t.Errorf("expected project test-app, got %s", c.Labels["com.docker.compose.project"])
		}
	}
}

func TestMockWorld_ContainerInspect(t *testing.T) {
	world, _ := setupTestWorld(t)

	resp, ok := world.ContainerInspect("mock-test-app-web-1")
	if !ok {
		t.Fatal("container not found")
	}
	if resp.State.Status != "running" {
		t.Errorf("state = %q, want running", resp.State.Status)
	}
	if resp.Config.Image != "nginx:latest" {
		t.Errorf("image = %q, want nginx:latest", resp.Config.Image)
	}
}

func TestMockWorld_NetworkIPConsistency(t *testing.T) {
	world, _ := setupTestWorld(t)

	// Get IPs from ContainerInspect
	resp, ok := world.ContainerInspect("mock-test-app-web-1")
	if !ok {
		t.Fatal("container not found")
	}
	inspectNets := resp.NetworkSettings.Networks
	frontendEP, hasFrontend := inspectNets["test-app_frontend"]
	if !hasFrontend {
		t.Fatal("container not on test-app_frontend network")
	}
	inspectIP := frontendEP.IPAddress
	inspectMAC := frontendEP.MacAddress

	// Get IPs from NetworkInspect
	netResp, ok := world.NetworkInspect("test-app_frontend")
	if !ok {
		t.Fatal("network not found")
	}
	netContainer, hasContainer := netResp.Containers["mock-test-app-web-1"]
	if !hasContainer {
		t.Fatal("container not in network inspect response")
	}

	// The critical check: IPs must match between ContainerInspect and NetworkInspect
	if netContainer.IPv4Address != inspectIP+"/16" {
		t.Errorf("NetworkInspect IP = %q, ContainerInspect IP = %q — MISMATCH",
			netContainer.IPv4Address, inspectIP)
	}
	if netContainer.MacAddress != inspectMAC {
		t.Errorf("NetworkInspect MAC = %q, ContainerInspect MAC = %q — MISMATCH",
			netContainer.MacAddress, inspectMAC)
	}

	// Verify exact IPs from mock.yaml
	if inspectIP != "172.20.0.2" {
		t.Errorf("web IP = %q, want 172.20.0.2 (from mock.yaml)", inspectIP)
	}
	if inspectMAC != "02:42:ac:14:00:02" {
		t.Errorf("web MAC = %q, want 02:42:ac:14:00:02 (from mock.yaml)", inspectMAC)
	}
}

func TestMockWorld_ContainerInspect_DataDriven(t *testing.T) {
	world, _ := setupTestWorld(t)

	// Check that command comes from mock.yaml
	resp, ok := world.ContainerInspect("mock-test-app-web-1")
	if !ok {
		t.Fatal("container not found")
	}
	if resp.Path != "nginx -g 'daemon off;'" {
		t.Errorf("path = %q, want nginx command from mock.yaml", resp.Path)
	}
	if resp.HostConfig.RestartPolicy.Name != "unless-stopped" {
		t.Errorf("restart = %q, want unless-stopped from mock.yaml", resp.HostConfig.RestartPolicy.Name)
	}

	// Check db health
	dbResp, ok := world.ContainerInspect("mock-test-app-db-1")
	if !ok {
		t.Fatal("db container not found")
	}
	if dbResp.Path != "postgres" {
		t.Errorf("db path = %q, want postgres from mock.yaml", dbResp.Path)
	}
}

func TestMockWorld_StandaloneContainers(t *testing.T) {
	world, _ := setupTestWorld(t)

	// Default fallback standalones should be present
	containers := world.ContainerList(true, "")
	foundStandalone := false
	for _, c := range containers {
		if c.ID == "mock-standalone-portainer" {
			foundStandalone = true
			if c.State != "running" {
				t.Errorf("portainer state = %q, want running", c.State)
			}
			break
		}
	}
	if !foundStandalone {
		t.Error("standalone portainer container not found")
	}

	// Standalone should not appear with project filter
	filtered := world.ContainerList(true, "test-app")
	for _, c := range filtered {
		if c.ID == "mock-standalone-portainer" {
			t.Error("standalone should not appear with project filter")
		}
	}
}

func TestMockWorld_NetworkList(t *testing.T) {
	world, _ := setupTestWorld(t)

	networks := world.NetworkList()
	if len(networks) == 0 {
		t.Fatal("expected networks")
	}

	foundBridge := false
	for _, n := range networks {
		if n.Name == "bridge" {
			foundBridge = true
		}
	}
	if !foundBridge {
		t.Error("missing bridge network")
	}
}
