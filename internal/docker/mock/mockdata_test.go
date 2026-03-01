package mock

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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
			"stack-010":     "running",
			"stack-013":     "exited",
			"stack-014":     "inactive",
		},
	}

	state := DefaultDevStateFromData(data)

	// All statuses come from data.stackStatuses
	if got := state.Get("03-monitoring"); got != "exited" {
		t.Errorf("03-monitoring = %q, want exited", got)
	}
	if got := state.Get("08-env-config"); got != "inactive" {
		t.Errorf("08-env-config = %q, want inactive", got)
	}
	if got := state.Get("stack-010"); got != "running" {
		t.Errorf("stack-010 = %q, want running", got)
	}
	if got := state.Get("stack-013"); got != "exited" {
		t.Errorf("stack-013 = %q, want exited", got)
	}
	if got := state.Get("stack-014"); got != "inactive" {
		t.Errorf("stack-014 = %q, want inactive", got)
	}

	// Unknown stacks default to "inactive"
	if got := state.Get("nonexistent"); got != "inactive" {
		t.Errorf("nonexistent = %q, want inactive", got)
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

// --- Log Template Tests ---

func TestExpandLogTemplate(t *testing.T) {
	baseTime := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	interval := 3 * time.Second

	tests := []struct {
		name     string
		input    string
		n        int
		image    string
		want     string
	}{
		{
			name:  "no variables",
			input: "plain text",
			n:     0, image: "nginx",
			want: "plain text",
		},
		{
			name:  "timestamp only",
			input: "{{.Timestamp}} hello",
			n:     0, image: "nginx",
			want: "2026-01-15T10:00:00.000Z hello",
		},
		{
			name:  "timestamp with N offset",
			input: "{{.Timestamp}} tick",
			n:     2, image: "nginx",
			want: "2026-01-15T10:00:06.000Z tick",
		},
		{
			name:  "N variable",
			input: "line {{.N}}",
			n:     5, image: "redis",
			want: "line 5",
		},
		{
			name:  "image variable",
			input: "{{.Image}} starting",
			n:     0, image: "postgres",
			want: "postgres starting",
		},
		{
			name:  "all variables",
			input: "{{.Timestamp}} {{.Image}} #{{.N}}",
			n:     1, image: "redis",
			want: "2026-01-15T10:00:03.000Z redis #1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandLogTemplate(tt.input, tt.n, baseTime, interval, tt.image)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseLogTemplates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log-templates.yaml")

	content := `base_time: "2026-01-15T10:00:00.000Z"

nginx:
  startup:
    - "/docker-entrypoint.sh: Configuration complete"
    - "{{.Timestamp}} [notice] 1#1: nginx/1.25.4"
  heartbeat:
    interval: 3s
    lines:
      - '172.17.0.2 - - [{{.Timestamp}}] "GET / HTTP/1.1" 200 1234'
  shutdown:
    - "{{.Timestamp}} [notice] 1#1: exiting"

redis:
  base_time: "2026-01-15T10:05:00.000Z"
  startup:
    - "1:C {{.Timestamp}} * Redis is starting"
  heartbeat:
    interval: 5s
    lines:
      - "1:M {{.Timestamp}} * DB saved on disk"
  shutdown:
    - "1:M {{.Timestamp}} # bye"

default:
  startup:
    - "{{.Image}} starting"
  heartbeat:
    interval: 3s
    lines:
      - "[INFO] OK"
  shutdown:
    - "[INFO] bye"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	templates := parseLogTemplates(path)

	// Nginx template
	nginx, ok := templates["nginx"]
	if !ok {
		t.Fatal("missing nginx template")
	}
	if len(nginx.Startup) != 2 {
		t.Errorf("nginx startup lines = %d, want 2", len(nginx.Startup))
	}
	if nginx.Startup[0] != "/docker-entrypoint.sh: Configuration complete" {
		t.Errorf("nginx startup[0] = %q", nginx.Startup[0])
	}
	if len(nginx.Heartbeat) != 1 {
		t.Errorf("nginx heartbeat lines = %d, want 1", len(nginx.Heartbeat))
	}
	if nginx.Interval != 3*time.Second {
		t.Errorf("nginx interval = %v, want 3s", nginx.Interval)
	}
	if len(nginx.Shutdown) != 1 {
		t.Errorf("nginx shutdown lines = %d, want 1", len(nginx.Shutdown))
	}
	// Nginx uses global base_time
	expectedTime := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	if !nginx.BaseTime.Equal(expectedTime) {
		t.Errorf("nginx base_time = %v, want %v", nginx.BaseTime, expectedTime)
	}

	// Redis has per-template base_time override
	redis, ok := templates["redis"]
	if !ok {
		t.Fatal("missing redis template")
	}
	expectedRedisTime := time.Date(2026, 1, 15, 10, 5, 0, 0, time.UTC)
	if !redis.BaseTime.Equal(expectedRedisTime) {
		t.Errorf("redis base_time = %v, want %v", redis.BaseTime, expectedRedisTime)
	}
	if redis.Interval != 5*time.Second {
		t.Errorf("redis interval = %v, want 5s", redis.Interval)
	}

	// Default template
	def, ok := templates["default"]
	if !ok {
		t.Fatal("missing default template")
	}
	if len(def.Startup) != 1 || def.Startup[0] != "{{.Image}} starting" {
		t.Errorf("default startup = %v", def.Startup)
	}
}

func TestParseLogTemplates_Missing(t *testing.T) {
	templates := parseLogTemplates("/nonexistent/log-templates.yaml")
	if len(templates) != 0 {
		t.Errorf("expected empty templates, got %d", len(templates))
	}
}

func TestGetServiceLogs_Resolution(t *testing.T) {
	dir := t.TempDir()

	// Create log-templates.yaml
	logTemplates := `base_time: "2026-01-15T10:00:00.000Z"
nginx:
  startup:
    - "nginx template startup"
  heartbeat:
    interval: 3s
    lines:
      - "nginx heartbeat"
default:
  startup:
    - "default startup"
  heartbeat:
    interval: 3s
    lines:
      - "default heartbeat"
`
	os.WriteFile(filepath.Join(dir, "log-templates.yaml"), []byte(logTemplates), 0644)

	// Create a stack with nginx service and per-service log override
	stackDir := filepath.Join(dir, "my-stack")
	os.MkdirAll(stackDir, 0755)
	os.WriteFile(filepath.Join(stackDir, "compose.yaml"), []byte(`services:
  web:
    image: nginx:latest
  api:
    image: node:20
  db:
    image: custom-image:latest
`), 0644)

	os.WriteFile(filepath.Join(stackDir, "mock.yaml"), []byte(`status: running
services:
  web:
    logs:
      startup:
        - "custom web startup"
`), 0644)

	data := BuildMockData(dir)

	// web has per-service override
	webLogs := data.GetServiceLogs("my-stack", "web")
	if len(webLogs.Startup) != 1 || webLogs.Startup[0] != "custom web startup" {
		t.Errorf("web logs startup = %v, want [custom web startup]", webLogs.Startup)
	}

	// api falls back to image template (node → not in templates, so falls through to default)
	apiLogs := data.GetServiceLogs("my-stack", "api")
	if len(apiLogs.Startup) != 1 || apiLogs.Startup[0] != "default startup" {
		t.Errorf("api logs startup = %v, want [default startup]", apiLogs.Startup)
	}

	// db has unknown image, falls through to default
	dbLogs := data.GetServiceLogs("my-stack", "db")
	if len(dbLogs.Startup) != 1 || dbLogs.Startup[0] != "default startup" {
		t.Errorf("db logs startup = %v, want [default startup]", dbLogs.Startup)
	}
}

func TestGetServiceLogs_ImageTemplate(t *testing.T) {
	dir := t.TempDir()

	logTemplates := `nginx:
  startup:
    - "nginx starting"
  heartbeat:
    interval: 4s
    lines:
      - "nginx heartbeat"
`
	os.WriteFile(filepath.Join(dir, "log-templates.yaml"), []byte(logTemplates), 0644)

	stackDir := filepath.Join(dir, "test")
	os.MkdirAll(stackDir, 0755)
	os.WriteFile(filepath.Join(stackDir, "compose.yaml"), []byte(`services:
  web:
    image: nginx:1.25
`), 0644)

	data := BuildMockData(dir)

	logs := data.GetServiceLogs("test", "web")
	if len(logs.Startup) != 1 || logs.Startup[0] != "nginx starting" {
		t.Errorf("startup = %v, want [nginx starting]", logs.Startup)
	}
	if logs.Interval != 4*time.Second {
		t.Errorf("interval = %v, want 4s", logs.Interval)
	}
}

func TestServiceLogsYAML_HasContent(t *testing.T) {
	empty := serviceLogsYAML{}
	if empty.hasContent() {
		t.Error("empty should not have content")
	}

	withStartup := serviceLogsYAML{Startup: []string{"hello"}}
	if !withStartup.hasContent() {
		t.Error("with startup should have content")
	}

	withBaseTime := serviceLogsYAML{BaseTime: "2026-01-15T10:00:00.000Z"}
	if !withBaseTime.hasContent() {
		t.Error("with base_time should have content")
	}
}

func TestServiceLogsYAML_Resolve(t *testing.T) {
	sly := serviceLogsYAML{
		BaseTime: "2026-06-15T12:00:00.000Z",
		Startup:  []string{"start"},
		Heartbeat: heartbeatYAML{
			Interval: "5s",
			Lines:    []string{"tick"},
		},
		Shutdown: []string{"stop"},
	}

	sl := sly.resolve()
	if sl.Interval != 5*time.Second {
		t.Errorf("interval = %v, want 5s", sl.Interval)
	}
	expectedTime := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	if !sl.BaseTime.Equal(expectedTime) {
		t.Errorf("base_time = %v, want %v", sl.BaseTime, expectedTime)
	}
	if len(sl.Startup) != 1 || sl.Startup[0] != "start" {
		t.Errorf("startup = %v", sl.Startup)
	}
	if len(sl.Heartbeat) != 1 || sl.Heartbeat[0] != "tick" {
		t.Errorf("heartbeat = %v", sl.Heartbeat)
	}
	if len(sl.Shutdown) != 1 || sl.Shutdown[0] != "stop" {
		t.Errorf("shutdown = %v", sl.Shutdown)
	}
}

func TestExtractImageBaseName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"nginx:latest", "nginx"},
		{"nginx:1.25", "nginx"},
		{"redis:7-alpine", "redis"},
		{"grafana/grafana:latest", "grafana"},
		{"ghcr.io/home-assistant/home-assistant:stable", "home-assistant"},
		{"portainer/portainer-ce:latest", "portainer-ce"},
		{"custom-image", "custom-image"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := extractImageBaseName(tt.input); got != tt.want {
				t.Errorf("extractImageBaseName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
