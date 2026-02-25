package compose

import (
    "os"
    "path/filepath"
    "sync"
    "testing"
)

func TestComposeCacheUpdateGetImages(t *testing.T) {
    t.Parallel()

    c := NewComposeCache()
    c.Update("my-stack", map[string]ServiceData{
        "web":   {Image: "nginx:latest", ImageUpdatesCheck: true},
        "redis": {Image: "redis:7", ImageUpdatesCheck: true},
        "build": {Image: "", ImageUpdatesCheck: true}, // no image (build-only)
    })

    images := c.GetImages("my-stack")
    if len(images) != 2 {
        t.Fatalf("expected 2 images (skip empty), got %d", len(images))
    }
    if images["web"] != "nginx:latest" {
        t.Errorf("web image = %q", images["web"])
    }
    if images["redis"] != "redis:7" {
        t.Errorf("redis image = %q", images["redis"])
    }
}

func TestComposeCacheGetImagesNonexistent(t *testing.T) {
    t.Parallel()

    c := NewComposeCache()
    images := c.GetImages("nonexistent")
    if images != nil {
        t.Errorf("expected nil for nonexistent stack, got %v", images)
    }
}

func TestComposeCacheGetServiceDataCopy(t *testing.T) {
    t.Parallel()

    c := NewComposeCache()
    c.Update("stack", map[string]ServiceData{
        "svc": {Image: "nginx:latest", StatusIgnore: false},
    })

    // Get a copy and mutate it
    copy := c.GetServiceData("stack")
    sd := copy["svc"]
    sd.StatusIgnore = true
    copy["svc"] = sd

    // Original should be unchanged
    original := c.GetServiceData("stack")
    if original["svc"].StatusIgnore {
        t.Error("mutating copy affected internal state")
    }
}

func TestComposeCacheIsStatusIgnored(t *testing.T) {
    t.Parallel()

    c := NewComposeCache()
    c.Update("stack", map[string]ServiceData{
        "ignored": {Image: "nginx", StatusIgnore: true},
        "normal":  {Image: "redis", StatusIgnore: false},
    })

    if !c.IsStatusIgnored("stack", "ignored") {
        t.Error("expected ignored=true")
    }
    if c.IsStatusIgnored("stack", "normal") {
        t.Error("expected normal=false")
    }
    // Unknown service defaults to false
    if c.IsStatusIgnored("stack", "unknown") {
        t.Error("expected unknown service=false")
    }
    // Unknown stack defaults to false
    if c.IsStatusIgnored("nonexistent", "any") {
        t.Error("expected unknown stack=false")
    }
}

func TestComposeCacheImageUpdatesEnabled(t *testing.T) {
    t.Parallel()

    c := NewComposeCache()
    c.Update("stack", map[string]ServiceData{
        "checked":   {Image: "nginx", ImageUpdatesCheck: true},
        "unchecked": {Image: "redis", ImageUpdatesCheck: false},
    })

    if !c.ImageUpdatesEnabled("stack", "checked") {
        t.Error("expected checked=true")
    }
    if c.ImageUpdatesEnabled("stack", "unchecked") {
        t.Error("expected unchecked=false")
    }
    // Unknown service defaults to true
    if !c.ImageUpdatesEnabled("stack", "unknown") {
        t.Error("expected unknown service=true (default)")
    }
    // Unknown stack defaults to true
    if !c.ImageUpdatesEnabled("nonexistent", "any") {
        t.Error("expected unknown stack=true (default)")
    }
}

func TestComposeCacheDelete(t *testing.T) {
    t.Parallel()

    c := NewComposeCache()
    c.Update("stack", map[string]ServiceData{
        "web": {Image: "nginx"},
    })

    c.Delete("stack")
    if images := c.GetImages("stack"); images != nil {
        t.Error("expected nil after delete")
    }
}

func TestComposeCacheGetAll(t *testing.T) {
    t.Parallel()

    c := NewComposeCache()
    c.Update("stack-a", map[string]ServiceData{
        "web": {Image: "nginx"},
    })
    c.Update("stack-b", map[string]ServiceData{
        "api": {Image: "node:20"},
    })

    all := c.GetAll()
    if len(all) != 2 {
        t.Fatalf("expected 2 stacks, got %d", len(all))
    }
    if all["stack-a"]["web"].Image != "nginx" {
        t.Error("stack-a web image mismatch")
    }

    // Verify deep copy: mutating returned value shouldn't affect cache
    all["stack-a"]["web"] = ServiceData{Image: "mutated"}
    original := c.GetAll()
    if original["stack-a"]["web"].Image != "nginx" {
        t.Error("deep copy failed: mutating returned map affected cache")
    }
}

func TestComposeCachePopulateFromDisk(t *testing.T) {
    t.Parallel()

    dir := t.TempDir()

    // Create a stack with a compose file
    stackDir := filepath.Join(dir, "my-stack")
    os.MkdirAll(stackDir, 0755)
    yaml := "services:\n  web:\n    image: nginx:latest\n  db:\n    image: postgres:16\n"
    os.WriteFile(filepath.Join(stackDir, "compose.yaml"), []byte(yaml), 0644)

    // Create a directory without a compose file (should be skipped)
    os.MkdirAll(filepath.Join(dir, "no-compose"), 0755)

    c := NewComposeCache()
    c.PopulateFromDisk(dir)

    images := c.GetImages("my-stack")
    if len(images) != 2 {
        t.Fatalf("expected 2 images, got %d", len(images))
    }
    if images["web"] != "nginx:latest" {
        t.Errorf("web = %q", images["web"])
    }

    // no-compose should not be in cache
    if c.GetImages("no-compose") != nil {
        t.Error("expected nil for directory without compose file")
    }
}

func TestComposeCacheConcurrent(t *testing.T) {
    t.Parallel()

    c := NewComposeCache()
    var wg sync.WaitGroup

    // 10 writers + 10 readers running concurrently
    for i := 0; i < 10; i++ {
        wg.Add(2)

        go func(id int) {
            defer wg.Done()
            name := "stack"
            for j := 0; j < 100; j++ {
                c.Update(name, map[string]ServiceData{
                    "svc": {Image: "nginx:latest"},
                })
                c.Delete(name)
            }
        }(i)

        go func(id int) {
            defer wg.Done()
            name := "stack"
            for j := 0; j < 100; j++ {
                c.GetImages(name)
                c.GetServiceData(name)
                c.IsStatusIgnored(name, "svc")
                c.ImageUpdatesEnabled(name, "svc")
                c.GetAll()
            }
        }(i)
    }

    wg.Wait()
}
