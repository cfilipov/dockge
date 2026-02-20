package compose

import (
    "log/slog"
    "os"
    "sync"
)

// ComposeCache stores extracted compose data for all stacks.
// Thread-safe via RWMutex. Only stores the fields we need, not the full YAML.
type ComposeCache struct {
    mu   sync.RWMutex
    data map[string]map[string]ServiceData // stackName -> serviceName -> data
}

// NewComposeCache creates an empty ComposeCache.
func NewComposeCache() *ComposeCache {
    return &ComposeCache{
        data: make(map[string]map[string]ServiceData),
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
        }
    }

    slog.Info("compose cache populated", "stacks", len(c.data))
}

// GetImages returns service→image map for a stack (replaces parseComposeImages).
func (c *ComposeCache) GetImages(stackName string) map[string]string {
    c.mu.RLock()
    defer c.mu.RUnlock()

    services, ok := c.data[stackName]
    if !ok {
        return nil
    }

    result := make(map[string]string, len(services))
    for svc, sd := range services {
        if sd.Image != "" {
            result[svc] = sd.Image
        }
    }
    return result
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

// GetAll returns a snapshot of the entire cache.
func (c *ComposeCache) GetAll() map[string]map[string]ServiceData {
    c.mu.RLock()
    defer c.mu.RUnlock()

    cp := make(map[string]map[string]ServiceData, len(c.data))
    for stackName, services := range c.data {
        svcCopy := make(map[string]ServiceData, len(services))
        for k, v := range services {
            svcCopy[k] = v
        }
        cp[stackName] = svcCopy
    }
    return cp
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

// Update replaces the cached data for a single stack.
func (c *ComposeCache) Update(stackName string, services map[string]ServiceData) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.data[stackName] = services
}

// Delete removes a stack from the cache.
func (c *ComposeCache) Delete(stackName string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    delete(c.data, stackName)
}
