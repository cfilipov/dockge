package compose

import (
    "os"
    "path/filepath"
    "testing"
)

func TestParseFile(t *testing.T) {
    // web-app: nginx (image updates changelog label, no dockge control labels),
    //          redis (status.ignore=true)
    data := ParseFile("/opt/stacks/01-web-app/compose.yaml")
    if len(data) != 2 {
        t.Fatalf("web-app: expected 2 services, got %d", len(data))
    }
    if nginx, ok := data["nginx"]; !ok {
        t.Error("web-app: missing nginx")
    } else {
        if nginx.Image != "nginx:latest" {
            t.Errorf("nginx.Image = %q", nginx.Image)
        }
        if nginx.StatusIgnore {
            t.Error("nginx.StatusIgnore should be false")
        }
    }
    if redis, ok := data["redis"]; !ok {
        t.Error("web-app: missing redis")
    } else {
        if redis.Image != "redis:alpine" {
            t.Errorf("redis.Image = %q", redis.Image)
        }
        if !redis.StatusIgnore {
            t.Error("redis.StatusIgnore should be true")
        }
    }

    // monitoring: grafana (imageupdates.check=false)
    data = ParseFile("/opt/stacks/03-monitoring/compose.yaml")
    if grafana, ok := data["grafana"]; !ok {
        t.Error("monitoring: missing grafana")
    } else {
        if grafana.ImageUpdatesCheck {
            t.Error("grafana.ImageUpdatesCheck should be false")
        }
    }
}

func TestParseYAML(t *testing.T) {
    yaml := `services:
  app:
    image: myapp:v2
    labels:
      dockge.status.ignore: "true"
      dockge.imageupdates.check: "false"
  db:
    image: postgres:16
`
    data := ParseYAML(yaml)

    app := data["app"]
    if app.Image != "myapp:v2" {
        t.Errorf("app.Image = %q", app.Image)
    }
    if !app.StatusIgnore {
        t.Error("app.StatusIgnore should be true")
    }
    if app.ImageUpdatesCheck {
        t.Error("app.ImageUpdatesCheck should be false")
    }

    db := data["db"]
    if db.Image != "postgres:16" {
        t.Errorf("db.Image = %q", db.Image)
    }
    if db.StatusIgnore {
        t.Error("db.StatusIgnore should be false")
    }
    if !db.ImageUpdatesCheck {
        t.Error("db.ImageUpdatesCheck should be true (default)")
    }
}

func TestParseYAMLLabelsBeforeImage(t *testing.T) {
    yaml := `services:
  svc:
    labels:
      dockge.imageupdates.check: "false"
    image: nginx:latest
`
    data := ParseYAML(yaml)
    svc := data["svc"]
    if svc.Image != "nginx:latest" {
        t.Errorf("svc.Image = %q", svc.Image)
    }
    if svc.ImageUpdatesCheck {
        t.Error("svc.ImageUpdatesCheck should be false")
    }
}

func TestParseYAMLNoLabels(t *testing.T) {
    yaml := `services:
  web:
    image: nginx:alpine
    ports:
      - "80:80"
`
    data := ParseYAML(yaml)
    web := data["web"]
    if web.Image != "nginx:alpine" {
        t.Errorf("web.Image = %q", web.Image)
    }
    if web.StatusIgnore {
        t.Error("web.StatusIgnore should be false")
    }
    if !web.ImageUpdatesCheck {
        t.Error("web.ImageUpdatesCheck should be true (default)")
    }
}

func TestParseYAMLMultipleTopLevelKeys(t *testing.T) {
    yaml := `version: "3"
services:
  app:
    image: myapp:v1
networks:
  default:
    driver: bridge
`
    data := ParseYAML(yaml)
    if len(data) != 1 {
        t.Fatalf("expected 1 service, got %d", len(data))
    }
    if data["app"].Image != "myapp:v1" {
        t.Errorf("app.Image = %q", data["app"].Image)
    }
}

func TestParseYAMLEmpty(t *testing.T) {
    t.Parallel()
    data := ParseYAML("")
    if len(data) != 0 {
        t.Errorf("expected 0 services for empty input, got %d", len(data))
    }
}

func TestParseYAMLNoServicesKey(t *testing.T) {
    t.Parallel()
    yaml := `version: "3"
networks:
  default:
    driver: bridge
`
    data := ParseYAML(yaml)
    if len(data) != 0 {
        t.Errorf("expected 0 services when no services key, got %d", len(data))
    }
}

func TestParseYAMLNoImage(t *testing.T) {
    t.Parallel()
    yaml := `services:
  builder:
    build:
      context: .
      dockerfile: Dockerfile
`
    data := ParseYAML(yaml)
    if len(data) != 1 {
        t.Fatalf("expected 1 service, got %d", len(data))
    }
    if data["builder"].Image != "" {
        t.Errorf("expected empty image for build-only service, got %q", data["builder"].Image)
    }
    // Should still have default label values
    if !data["builder"].ImageUpdatesCheck {
        t.Error("ImageUpdatesCheck should default to true")
    }
}

func TestParseYAMLServiceNameVariants(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name    string
        svcName string
    }{
        {"hyphenated", "my-service"},
        {"underscored", "my_service"},
        {"dotted", "my.service"},
        {"simple", "web"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            yaml := "services:\n  " + tt.svcName + ":\n    image: alpine:latest\n"
            data := ParseYAML(yaml)
            if _, ok := data[tt.svcName]; !ok {
                t.Errorf("service %q not found in parsed output", tt.svcName)
            }
        })
    }
}

func TestParseYAMLQuotedLabelValues(t *testing.T) {
    t.Parallel()
    yaml := `services:
  app:
    image: myapp:v1
    labels:
      dockge.status.ignore: 'true'
      dockge.imageupdates.check: "false"
`
    data := ParseYAML(yaml)
    if !data["app"].StatusIgnore {
        t.Error("expected StatusIgnore=true with single-quoted value")
    }
    if data["app"].ImageUpdatesCheck {
        t.Error("expected ImageUpdatesCheck=false with double-quoted value")
    }
}

func TestParseYAMLCommentsAndBlankLines(t *testing.T) {
    t.Parallel()
    yaml := `# Top comment
services:
  # Service comment
  web:
    image: nginx:latest

    # Port comment
    ports:
      - "80:80"
`
    data := ParseYAML(yaml)
    if len(data) != 1 {
        t.Fatalf("expected 1 service, got %d", len(data))
    }
    if data["web"].Image != "nginx:latest" {
        t.Errorf("web.Image = %q", data["web"].Image)
    }
}

func TestFindComposeFile(t *testing.T) {
    t.Parallel()

    t.Run("compose.yaml", func(t *testing.T) {
        t.Parallel()
        dir := t.TempDir()
        stackDir := filepath.Join(dir, "my-stack")
        os.MkdirAll(stackDir, 0755)
        os.WriteFile(filepath.Join(stackDir, "compose.yaml"), []byte("services: {}"), 0644)

        got := FindComposeFile(dir, "my-stack")
        if got == "" {
            t.Fatal("expected non-empty path")
        }
        if filepath.Base(got) != "compose.yaml" {
            t.Errorf("expected compose.yaml, got %s", filepath.Base(got))
        }
    })

    t.Run("docker-compose.yml fallback", func(t *testing.T) {
        t.Parallel()
        dir := t.TempDir()
        stackDir := filepath.Join(dir, "old-stack")
        os.MkdirAll(stackDir, 0755)
        os.WriteFile(filepath.Join(stackDir, "docker-compose.yml"), []byte("services: {}"), 0644)

        got := FindComposeFile(dir, "old-stack")
        if filepath.Base(got) != "docker-compose.yml" {
            t.Errorf("expected docker-compose.yml, got %s", filepath.Base(got))
        }
    })

    t.Run("priority order", func(t *testing.T) {
        t.Parallel()
        dir := t.TempDir()
        stackDir := filepath.Join(dir, "multi")
        os.MkdirAll(stackDir, 0755)
        // Create both — compose.yaml should win
        os.WriteFile(filepath.Join(stackDir, "compose.yaml"), []byte("a"), 0644)
        os.WriteFile(filepath.Join(stackDir, "docker-compose.yml"), []byte("b"), 0644)

        got := FindComposeFile(dir, "multi")
        if filepath.Base(got) != "compose.yaml" {
            t.Errorf("expected compose.yaml (higher priority), got %s", filepath.Base(got))
        }
    })

    t.Run("missing directory", func(t *testing.T) {
        t.Parallel()
        dir := t.TempDir()
        got := FindComposeFile(dir, "nonexistent")
        if got != "" {
            t.Errorf("expected empty for missing dir, got %q", got)
        }
    })

    t.Run("empty directory", func(t *testing.T) {
        t.Parallel()
        dir := t.TempDir()
        os.MkdirAll(filepath.Join(dir, "empty"), 0755)
        got := FindComposeFile(dir, "empty")
        if got != "" {
            t.Errorf("expected empty for dir with no compose file, got %q", got)
        }
    })
}

func TestParseFileNonexistent(t *testing.T) {
    t.Parallel()
    data := ParseFile("/nonexistent/path/compose.yaml")
    if data != nil {
        t.Error("expected nil for nonexistent file")
    }
}

func FuzzParseYAML(f *testing.F) {
    // Seed corpus with valid and edge-case inputs
    f.Add("services:\n  web:\n    image: nginx:latest\n")
    f.Add("")
    f.Add("services:\n")
    f.Add("services:\n  svc:\n    labels:\n      dockge.status.ignore: \"true\"\n")
    f.Add("not yaml at all")
    f.Add("services:\n  a:\n    image: x\n  b:\n    image: y\n")

    f.Fuzz(func(t *testing.T, input string) {
        // Should not panic
        ParseYAML(input)
    })
}
