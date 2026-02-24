package docker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMockYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mock.yaml")

	content := `status: running
services:
  redis:
    state: exited
  nginx:
    running_image: "nginx:1.24"
    update_available: true
  db:
    health: unhealthy
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	mo := parseMockYAML(path)

	if mo.status != "running" {
		t.Errorf("status = %q, want %q", mo.status, "running")
	}
	if len(mo.services) != 3 {
		t.Fatalf("services count = %d, want 3", len(mo.services))
	}
	if mo.services["redis"].state != "exited" {
		t.Errorf("redis.state = %q, want %q", mo.services["redis"].state, "exited")
	}
	if mo.services["nginx"].runningImage != "nginx:1.24" {
		t.Errorf("nginx.runningImage = %q, want %q", mo.services["nginx"].runningImage, "nginx:1.24")
	}
	if !mo.services["nginx"].updateAvailable {
		t.Error("nginx.updateAvailable = false, want true")
	}
	if mo.services["db"].health != "unhealthy" {
		t.Errorf("db.health = %q, want %q", mo.services["db"].health, "unhealthy")
	}
}

func TestParseMockYAML_Missing(t *testing.T) {
	mo := parseMockYAML("/nonexistent/mock.yaml")
	if mo.status != "" {
		t.Errorf("status = %q, want empty", mo.status)
	}
	if mo.services != nil {
		t.Errorf("services = %v, want nil", mo.services)
	}
}

func TestParseMockYAML_StatusOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mock.yaml")

	if err := os.WriteFile(path, []byte("status: exited\n"), 0644); err != nil {
		t.Fatal(err)
	}

	mo := parseMockYAML(path)
	if mo.status != "exited" {
		t.Errorf("status = %q, want %q", mo.status, "exited")
	}
	if len(mo.services) != 0 {
		t.Errorf("services count = %d, want 0", len(mo.services))
	}
}

func TestParseComposeForMock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")

	content := `services:
  app:
    image: node:20-alpine
    command: node server.js
    networks:
      - frontend
      - backend
  db:
    image: postgres:16
    volumes:
      - dbdata:/var/lib/postgresql/data
    networks:
      - backend
  cache:
    image: redis:7
    networks:
      - backend

volumes:
  dbdata:

networks:
  frontend:
  backend:
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cd := parseComposeForMock(path)

	// Services
	if len(cd.services) != 3 {
		t.Fatalf("services count = %d, want 3", len(cd.services))
	}
	if cd.services[0].name != "app" || cd.services[0].image != "node:20-alpine" {
		t.Errorf("service[0] = %+v, want app/node:20-alpine", cd.services[0])
	}
	if cd.services[1].name != "db" || cd.services[1].image != "postgres:16" {
		t.Errorf("service[1] = %+v, want db/postgres:16", cd.services[1])
	}
	if cd.services[2].name != "cache" || cd.services[2].image != "redis:7" {
		t.Errorf("service[2] = %+v, want cache/redis:7", cd.services[2])
	}

	// Networks
	if len(cd.services[0].networks) != 2 {
		t.Errorf("app networks = %v, want [frontend backend]", cd.services[0].networks)
	}
	if len(cd.services[1].networks) != 1 || cd.services[1].networks[0] != "backend" {
		t.Errorf("db networks = %v, want [backend]", cd.services[1].networks)
	}

	// Volumes
	if len(cd.services[1].volumes) != 1 {
		t.Fatalf("db volumes count = %d, want 1", len(cd.services[1].volumes))
	}
	vol := cd.services[1].volumes[0]
	if vol.name != "dbdata" || vol.destination != "/var/lib/postgresql/data" || !vol.isNamed {
		t.Errorf("db volume = %+v, want dbdata:/var/lib/postgresql/data (named)", vol)
	}

	// Top-level
	if len(cd.networks) != 2 {
		t.Errorf("top-level networks = %v, want [frontend backend]", cd.networks)
	}
	if len(cd.volumes) != 1 || cd.volumes[0] != "dbdata" {
		t.Errorf("top-level volumes = %v, want [dbdata]", cd.volumes)
	}
}

func TestParseComposeForMock_BindVolume(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")

	content := `services:
  web:
    image: nginx:latest
    volumes:
      - ./html:/usr/share/nginx/html:ro
      - /var/run/docker.sock:/var/run/docker.sock

volumes: {}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cd := parseComposeForMock(path)
	if len(cd.services) != 1 {
		t.Fatalf("services count = %d, want 1", len(cd.services))
	}
	if len(cd.services[0].volumes) != 2 {
		t.Fatalf("volumes count = %d, want 2", len(cd.services[0].volumes))
	}
	// ./html is a bind mount, not a named volume
	v := cd.services[0].volumes[0]
	if v.isNamed {
		t.Error("./html should not be a named volume")
	}
	if !v.readOnly {
		t.Error("./html:....:ro should be readOnly")
	}
}

func TestBuildMockData(t *testing.T) {
	dir := t.TempDir()

	// Create a test stack
	stackDir := filepath.Join(dir, "my-stack")
	if err := os.MkdirAll(stackDir, 0755); err != nil {
		t.Fatal(err)
	}

	compose := `services:
  web:
    image: nginx:latest
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
`
	mockYAML := `status: running
services:
  web:
    running_image: "nginx:1.24"
    update_available: true
`
	os.WriteFile(filepath.Join(stackDir, "compose.yaml"), []byte(compose), 0644)
	os.WriteFile(filepath.Join(stackDir, "mock.yaml"), []byte(mockYAML), 0644)

	data := BuildMockData(dir)

	// Check images
	if _, ok := data.images["nginx:latest"]; !ok {
		t.Error("missing image nginx:latest")
	}
	if _, ok := data.images["nginx:1.24"]; !ok {
		t.Error("missing image nginx:1.24 (from mock.yaml running_image)")
	}
	if _, ok := data.images["postgres:16"]; !ok {
		t.Error("missing image postgres:16")
	}

	// Check standalone images
	if _, ok := data.images["portainer/portainer-ce:latest"]; !ok {
		t.Error("missing standalone image portainer/portainer-ce:latest")
	}

	// Check networks
	if _, ok := data.networks["my-stack_frontend"]; !ok {
		t.Error("missing network my-stack_frontend")
	}
	if _, ok := data.networks["my-stack_backend"]; !ok {
		t.Error("missing network my-stack_backend")
	}
	if _, ok := data.networks["bridge"]; !ok {
		t.Error("missing default network bridge")
	}

	// Check volumes
	if _, ok := data.volumes["my-stack_pgdata"]; !ok {
		t.Error("missing volume my-stack_pgdata")
	}

	// Check service states from mock.yaml
	if data.GetRunningImage("my-stack", "web") != "nginx:1.24" {
		t.Errorf("running image = %q, want nginx:1.24", data.GetRunningImage("my-stack", "web"))
	}
	if data.GetRunningImage("my-stack", "db") != "postgres:16" {
		t.Errorf("running image = %q, want postgres:16", data.GetRunningImage("my-stack", "db"))
	}

	// Check update flags
	if !data.HasUpdateAvailable("nginx:1.24") {
		t.Error("nginx:1.24 should have update available")
	}
	if data.HasUpdateAvailable("postgres:16") {
		t.Error("postgres:16 should not have update available")
	}

	// Check stack statuses
	if data.stackStatuses["my-stack"] != "running" {
		t.Errorf("stack status = %q, want running", data.stackStatuses["my-stack"])
	}

	// Check service networks
	if nets, ok := data.serviceNetworks["my-stack/web"]; !ok || len(nets) != 1 || nets[0] != "my-stack_frontend" {
		t.Errorf("web networks = %v, want [my-stack_frontend]", nets)
	}
	if nets, ok := data.serviceNetworks["my-stack/db"]; !ok || len(nets) != 1 || nets[0] != "my-stack_backend" {
		t.Errorf("db networks = %v, want [my-stack_backend]", nets)
	}

	// Check service volumes
	if mounts, ok := data.serviceVolumes["my-stack/db"]; !ok || len(mounts) != 1 {
		t.Fatalf("db volumes = %v, want 1 mount", mounts)
	} else {
		if mounts[0].name != "my-stack_pgdata" {
			t.Errorf("db volume name = %q, want my-stack_pgdata", mounts[0].name)
		}
		if mounts[0].destination != "/var/lib/postgresql/data" {
			t.Errorf("db volume dest = %q, want /var/lib/postgresql/data", mounts[0].destination)
		}
	}

	// Check dangling images
	if _, ok := data.images["<dangling:1>"]; !ok {
		t.Error("missing dangling image 1")
	}
	if _, ok := data.images["<dangling:2>"]; !ok {
		t.Error("missing dangling image 2")
	}
}

func TestBuildMockData_DefaultNetwork(t *testing.T) {
	dir := t.TempDir()

	// Stack without explicit networks should get a default network
	stackDir := filepath.Join(dir, "simple")
	os.MkdirAll(stackDir, 0755)
	os.WriteFile(filepath.Join(stackDir, "compose.yaml"), []byte(`services:
  web:
    image: nginx:latest
`), 0644)

	data := BuildMockData(dir)

	if _, ok := data.networks["simple_default"]; !ok {
		t.Error("missing default network simple_default")
	}
	if nets := data.serviceNetworks["simple/web"]; len(nets) != 1 || nets[0] != "simple_default" {
		t.Errorf("web networks = %v, want [simple_default]", nets)
	}
}

func TestMockData_GetServiceState(t *testing.T) {
	data := &MockData{
		serviceStates: map[string]string{
			"stack/worker": "exited",
		},
	}

	if got := data.GetServiceState("stack", "app", "running"); got != "running" {
		t.Errorf("app state = %q, want running", got)
	}
	if got := data.GetServiceState("stack", "worker", "running"); got != "exited" {
		t.Errorf("worker state = %q, want exited", got)
	}
	if got := data.GetServiceState("stack", "app", "exited"); got != "exited" {
		t.Errorf("app state (exited stack) = %q, want exited", got)
	}
}

func TestMockData_GetServiceHealth(t *testing.T) {
	data := &MockData{
		serviceHealth: map[string]string{
			"stack/db": "unhealthy",
		},
	}

	if got := data.GetServiceHealth("stack", "db"); got != "unhealthy" {
		t.Errorf("db health = %q, want unhealthy", got)
	}
	if got := data.GetServiceHealth("stack", "app"); got != "" {
		t.Errorf("app health = %q, want empty", got)
	}
}

func TestMockData_ParseContainerKey(t *testing.T) {
	data := &MockData{
		serviceImages: map[string]string{
			"01-web-app/nginx": "nginx:latest",
			"01-web-app/redis": "redis:7",
			"simple/web":       "nginx:latest",
		},
	}

	tests := []struct {
		id          string
		wantStack   string
		wantService string
		wantOk      bool
	}{
		{"mock-01-web-app-nginx-1", "01-web-app", "nginx", true},
		{"mock-01-web-app-redis-1", "01-web-app", "redis", true},
		{"mock-simple-web-1", "simple", "web", true},
		{"mock-unknown-svc-1", "", "", false},
	}

	for _, tt := range tests {
		stack, svc, ok := data.parseContainerKey(tt.id)
		if stack != tt.wantStack || svc != tt.wantService || ok != tt.wantOk {
			t.Errorf("parseContainerKey(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tt.id, stack, svc, ok, tt.wantStack, tt.wantService, tt.wantOk)
		}
	}
}

func TestDefaultDevStateFromData(t *testing.T) {
	data := &MockData{
		stackStatuses: map[string]string{
			"03-monitoring": "exited",
			"08-env-config": "inactive",
		},
	}

	state := DefaultDevStateFromData(data)

	// Featured stacks with mock.yaml overrides
	if got := state.Get("03-monitoring"); got != "exited" {
		t.Errorf("03-monitoring = %q, want exited", got)
	}
	if got := state.Get("08-env-config"); got != "inactive" {
		t.Errorf("08-env-config = %q, want inactive", got)
	}

	// Filler stacks still get default distribution
	if got := state.Get("stack-010"); got != "running" {
		t.Errorf("stack-010 = %q, want running (10%%5=0)", got)
	}
	if got := state.Get("stack-013"); got != "exited" {
		t.Errorf("stack-013 = %q, want exited (13%%5=3)", got)
	}
	if got := state.Get("stack-014"); got != "inactive" {
		t.Errorf("stack-014 = %q, want inactive (14%%5=4)", got)
	}
}

func TestMockState_Reset(t *testing.T) {
	defaults := map[string]string{
		"stack-a": "running",
		"stack-b": "exited",
	}
	state := NewMockStateFrom(defaults)

	// Mutate
	state.Set("stack-a", "exited")
	state.Set("stack-c", "running")
	if got := state.Get("stack-a"); got != "exited" {
		t.Fatalf("after mutation: stack-a = %q, want exited", got)
	}

	// Reset
	state.Reset()
	if got := state.Get("stack-a"); got != "running" {
		t.Errorf("after reset: stack-a = %q, want running", got)
	}
	if got := state.Get("stack-c"); got != "inactive" {
		t.Errorf("after reset: stack-c = %q, want inactive (removed)", got)
	}
}
