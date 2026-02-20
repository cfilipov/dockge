package models

import (
    "bytes"
    "encoding/json"
    "fmt"

    bolt "go.etcd.io/bbolt"

    "github.com/cfilipov/dockge/backend-go/internal/db"
)

// ImageUpdateStore manages cached image update check results.
type ImageUpdateStore struct {
    db *bolt.DB
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
func (s *ImageUpdateStore) StackHasUpdates() (map[string]bool, error) {
    entries, err := s.GetAll()
    if err != nil {
        return nil, err
    }
    result := make(map[string]bool)
    for _, e := range entries {
        if e.HasUpdate {
            result[e.StackName] = true
        }
    }
    return result, nil
}

// Upsert inserts or updates a single cache entry.
func (s *ImageUpdateStore) Upsert(stackName, serviceName, imageRef, localDigest, remoteDigest string, hasUpdate bool) error {
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
