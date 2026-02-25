package models

import (
    "bytes"
    "encoding/json"
    "fmt"
    "sync/atomic"

    bolt "go.etcd.io/bbolt"

    "github.com/cfilipov/dockge/internal/db"
)

// ImageUpdateStore manages cached image update check results.
// An in-memory cache (atomic pointer) avoids reading BoltDB on every broadcast.
type ImageUpdateStore struct {
    db           *bolt.DB
    cache        atomic.Pointer[map[string]bool] // lazily rebuilt, invalidated on writes
    serviceCache atomic.Pointer[map[string]bool] // "stack/service" → hasUpdate, invalidated on writes
}

func NewImageUpdateStore(database *bolt.DB) *ImageUpdateStore {
    return &ImageUpdateStore{db: database}
}

// ImageUpdateEntry represents a cached image update check result.
type ImageUpdateEntry struct {
    StackName   string `json:"stackName"`
    ServiceName string `json:"serviceName"`
    HasUpdate   bool   `json:"hasUpdate"`
}

// imageUpdateRecord is the full stored record (superset of ImageUpdateEntry).
type imageUpdateRecord struct {
    StackName    string `json:"stackName"`
    ServiceName  string `json:"serviceName"`
    ImageRef     string `json:"imageRef,omitempty"`
    LocalDigest  string `json:"localDigest,omitempty"`
    RemoteDigest string `json:"remoteDigest,omitempty"`
    HasUpdate    bool   `json:"hasUpdate"`
    LastChecked  int64  `json:"lastChecked,omitempty"`
}

// compoundKey returns "stackName/serviceName" as the bbolt key.
func compoundKey(stackName, serviceName string) []byte {
    return []byte(stackName + "/" + serviceName)
}

// stackPrefix returns "stackName/" for prefix scanning.
func stackPrefix(stackName string) []byte {
    return []byte(stackName + "/")
}

// GetAll returns all cached image update entries.
func (s *ImageUpdateStore) GetAll() ([]ImageUpdateEntry, error) {
    var entries []ImageUpdateEntry
    err := s.db.View(func(tx *bolt.Tx) error {
        return tx.Bucket(db.BucketImageUpdates).ForEach(func(k, v []byte) error {
            var rec imageUpdateRecord
            if err := json.Unmarshal(v, &rec); err != nil {
                return fmt.Errorf("unmarshal image update %q: %w", string(k), err)
            }
            entries = append(entries, ImageUpdateEntry{
                StackName:   rec.StackName,
                ServiceName: rec.ServiceName,
                HasUpdate:   rec.HasUpdate,
            })
            return nil
        })
    })
    if err != nil {
        return nil, err
    }
    return entries, nil
}

// StackHasUpdates returns a map of stack name → true if any service has an update.
// Uses an in-memory cache that is lazily rebuilt on first read and invalidated
// on Upsert/DeleteForStack/DeleteService — avoiding BoltDB I/O on every broadcast.
func (s *ImageUpdateStore) StackHasUpdates() (map[string]bool, error) {
    if cached := s.cache.Load(); cached != nil {
        return *cached, nil
    }
    return s.rebuildCache()
}

// rebuildCache reads all entries from BoltDB and populates the in-memory cache.
func (s *ImageUpdateStore) rebuildCache() (map[string]bool, error) {
    entries, err := s.GetAll()
    if err != nil {
        return nil, err
    }
    result := make(map[string]bool, len(entries))
    for _, e := range entries {
        if e.HasUpdate {
            result[e.StackName] = true
        }
    }
    s.cache.Store(&result)
    return result, nil
}

// invalidateCache clears both in-memory caches, forcing a rebuild on next read.
func (s *ImageUpdateStore) invalidateCache() {
    s.cache.Store(nil)
    s.serviceCache.Store(nil)
}

// AllServiceUpdates returns a map of "stackName/serviceName" → true for services
// with available image updates. Uses an in-memory cache (same invalidation as
// StackHasUpdates) to avoid repeated BoltDB scans.
func (s *ImageUpdateStore) AllServiceUpdates() (map[string]bool, error) {
    if cached := s.serviceCache.Load(); cached != nil {
        return *cached, nil
    }
    return s.rebuildServiceCache()
}

// rebuildServiceCache reads all entries from BoltDB and populates the per-service cache.
func (s *ImageUpdateStore) rebuildServiceCache() (map[string]bool, error) {
    entries, err := s.GetAll()
    if err != nil {
        return nil, err
    }
    result := make(map[string]bool, len(entries))
    for _, e := range entries {
        if e.HasUpdate {
            result[e.StackName+"/"+e.ServiceName] = true
        }
    }
    s.serviceCache.Store(&result)
    return result, nil
}

// Upsert inserts or updates a single cache entry.
func (s *ImageUpdateStore) Upsert(stackName, serviceName, imageRef, localDigest, remoteDigest string, hasUpdate bool) error {
    defer s.invalidateCache()
    return s.db.Update(func(tx *bolt.Tx) error {
        rec := imageUpdateRecord{
            StackName:    stackName,
            ServiceName:  serviceName,
            ImageRef:     imageRef,
            LocalDigest:  localDigest,
            RemoteDigest: remoteDigest,
            HasUpdate:    hasUpdate,
        }
        data, err := json.Marshal(&rec)
        if err != nil {
            return fmt.Errorf("marshal image update: %w", err)
        }
        return tx.Bucket(db.BucketImageUpdates).Put(compoundKey(stackName, serviceName), data)
    })
}

// DeleteForStack removes all cache entries for a stack.
func (s *ImageUpdateStore) DeleteForStack(stackName string) error {
    defer s.invalidateCache()
    prefix := stackPrefix(stackName)
    return s.db.Update(func(tx *bolt.Tx) error {
        b := tx.Bucket(db.BucketImageUpdates)
        c := b.Cursor()
        for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
            if err := b.Delete(k); err != nil {
                return err
            }
        }
        return nil
    })
}

// DeleteService removes a single service's cache entry.
func (s *ImageUpdateStore) DeleteService(stackName, serviceName string) error {
    defer s.invalidateCache()
    return s.db.Update(func(tx *bolt.Tx) error {
        return tx.Bucket(db.BucketImageUpdates).Delete(compoundKey(stackName, serviceName))
    })
}

// SeedFromMock clears all existing image update entries and writes the given
// flags ("stackName/serviceName" → hasUpdate) into BoltDB. Used in mock mode
// to ensure BoltDB state matches mock.yaml on startup and mock reset.
func (s *ImageUpdateStore) SeedFromMock(flags map[string]bool) error {
    defer s.invalidateCache()
    return s.db.Update(func(tx *bolt.Tx) error {
        b := tx.Bucket(db.BucketImageUpdates)
        // Clear all existing entries
        c := b.Cursor()
        for k, _ := c.First(); k != nil; k, _ = c.Next() {
            if err := b.Delete(k); err != nil {
                return err
            }
        }
        // Write mock flags
        for key, hasUpdate := range flags {
            parts := bytes.SplitN([]byte(key), []byte("/"), 2)
            if len(parts) != 2 {
                continue
            }
            rec := imageUpdateRecord{
                StackName:   string(parts[0]),
                ServiceName: string(parts[1]),
                HasUpdate:   hasUpdate,
            }
            data, err := json.Marshal(&rec)
            if err != nil {
                return fmt.Errorf("marshal mock image update: %w", err)
            }
            if err := b.Put([]byte(key), data); err != nil {
                return err
            }
        }
        return nil
    })
}

// ServiceUpdatesForStack returns a map of service name → has_update for a given stack.
func (s *ImageUpdateStore) ServiceUpdatesForStack(stackName string) (map[string]bool, error) {
    prefix := stackPrefix(stackName)
    result := make(map[string]bool)
    err := s.db.View(func(tx *bolt.Tx) error {
        c := tx.Bucket(db.BucketImageUpdates).Cursor()
        for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
            var rec imageUpdateRecord
            if err := json.Unmarshal(v, &rec); err != nil {
                return fmt.Errorf("unmarshal image update %q: %w", string(k), err)
            }
            result[rec.ServiceName] = rec.HasUpdate
        }
        return nil
    })
    if err != nil {
        return nil, err
    }
    return result, nil
}
