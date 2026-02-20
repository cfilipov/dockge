package compose

import "testing"

func TestParseFile(t *testing.T) {
    // web-app: nginx (image updates changelog label, no dockge control labels),
    //          redis (status.ignore=true)
    data := ParseFile("/opt/stacks/web-app/compose.yaml")
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
    data = ParseFile("/opt/stacks/monitoring/compose.yaml")
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
