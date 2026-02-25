package compose

import (
    "log/slog"
    "os"
    "sync"
)

// ComposeCache stores extracted compose data for all stacks.
// Thread-safe via RWMutex. Only stores the fields we need, not the full YAML.
type ComposeCache struct {
    mu     sync.RWMutex
    data   map[string]map[string]ServiceData // stackName -> serviceName -> data
    images map[string]map[string]string      // stackName -> service -> image, rebuilt on Set
}

// NewComposeCache creates an empty ComposeCache.
func NewComposeCache() *ComposeCache {
    return &ComposeCache{
        data:   make(map[string]map[string]ServiceData),
        images: make(map[string]map[string]string),
    }
}

// PopulateFromDisk scans the stacks directory and parses all compose files.
// Called once at startup before the watcher starts.
func (c *ComposeCache) PopulateFromDisk(stacksDir string) {
    entries, err := os.ReadDir(stacksDir)
    if err != nil {
        slog.Warn("compose cache: scan stacks dir", "err", err, "dir", stacksDir)
        return
    }

    c.mu.Lock()
    defer c.mu.Unlock()

    for _, entry := range entries {
        if !entry.IsDir() {
            continue
        }
        name := entry.Name()
        path := FindComposeFile(stacksDir, name)
        if path == "" {
            continue
        }
        services := ParseFile(path)
        if services != nil {
            c.data[name] = services
            c.images[name] = buildImagesMap(services)
        }
    }

    slog.Info("compose cache populated", "stacks", len(c.data))
}

// GetImages returns the cached service→image map for a stack. The returned map
// is owned by the cache — callers must NOT mutate it. Returns nil if the stack
// is not in the cache. Zero allocations.
func (c *ComposeCache) GetImages(stackName string) map[string]string {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.images[stackName]
}

// GetServiceData returns all service data for a stack.
func (c *ComposeCache) GetServiceData(stackName string) map[string]ServiceData {
    c.mu.RLock()
    defer c.mu.RUnlock()

    services, ok := c.data[stackName]
    if !ok {
        return nil
    }

    // Return a copy to avoid races
    cp := make(map[string]ServiceData, len(services))
    for k, v := range services {
        cp[k] = v
    }
    return cp
}

// BuildIgnoreMap returns stackName → serviceName → true for all services with
// StatusIgnore=true. Reads directly under RLock, avoiding the deep copy of GetAll().
func (c *ComposeCache) BuildIgnoreMap() map[string]map[string]bool {
    c.mu.RLock()
    defer c.mu.RUnlock()

    result := make(map[string]map[string]bool)
    for stackName, services := range c.data {
        for svcName, sd := range services {
            if sd.StatusIgnore {
                if result[stackName] == nil {
                    result[stackName] = make(map[string]bool)
                }
                result[stackName][svcName] = true
            }
        }
    }
    return result
}

// IsStatusIgnored returns true if a service has dockge.status.ignore=true.
func (c *ComposeCache) IsStatusIgnored(stackName, serviceName string) bool {
    c.mu.RLock()
    defer c.mu.RUnlock()

    if services, ok := c.data[stackName]; ok {
        if sd, ok := services[serviceName]; ok {
            return sd.StatusIgnore
        }
    }
    return false
}

// ImageUpdatesEnabled returns false if the service has imageupdates.check=false.
func (c *ComposeCache) ImageUpdatesEnabled(stackName, serviceName string) bool {
    c.mu.RLock()
    defer c.mu.RUnlock()

    if services, ok := c.data[stackName]; ok {
        if sd, ok := services[serviceName]; ok {
            return sd.ImageUpdatesCheck
        }
    }
    return true // default: enabled
}

// Update replaces the cached data for a single stack and rebuilds its images map.
func (c *ComposeCache) Update(stackName string, services map[string]ServiceData) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.data[stackName] = services
    c.images[stackName] = buildImagesMap(services)
}

// Delete removes a stack from the cache.
func (c *ComposeCache) Delete(stackName string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    delete(c.data, stackName)
    delete(c.images, stackName)
}

// buildImagesMap extracts a service→image map from service data, skipping
// build-only services (empty image field).
func buildImagesMap(services map[string]ServiceData) map[string]string {
    m := make(map[string]string, len(services))
    for svc, sd := range services {
        if sd.Image != "" {
            m[svc] = sd.Image
        }
    }
    return m
}
