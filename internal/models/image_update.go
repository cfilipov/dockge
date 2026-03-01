package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/cfilipov/dockge/internal/db"
)

// ImageUpdateStore manages cached image update check results in BoltDB.
// No in-memory cache — BoltDB is memory-mapped so reads are ~0.5ms.
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
// Scans BoltDB directly (~0.5ms, memory-mapped).
func (s *ImageUpdateStore) StackHasUpdates() (map[string]bool, error) {
	result := make(map[string]bool)
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(db.BucketImageUpdates).ForEach(func(k, v []byte) error {
			var rec imageUpdateRecord
			if err := json.Unmarshal(v, &rec); err != nil {
				return nil // skip corrupt entries
			}
			if rec.HasUpdate {
				result[rec.StackName] = true
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// AllServiceUpdates returns a map of "stackName/serviceName" → true for services
// with available image updates. Scans BoltDB directly.
func (s *ImageUpdateStore) AllServiceUpdates() (map[string]bool, error) {
	result := make(map[string]bool)
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(db.BucketImageUpdates).ForEach(func(k, v []byte) error {
			var rec imageUpdateRecord
			if err := json.Unmarshal(v, &rec); err != nil {
				return nil // skip corrupt entries
			}
			if rec.HasUpdate {
				result[rec.StackName+"/"+rec.ServiceName] = true
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
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

// DeleteService removes a single service's cache entry.
func (s *ImageUpdateStore) DeleteService(stackName, serviceName string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(db.BucketImageUpdates).Delete(compoundKey(stackName, serviceName))
	})
}

// SeedFromMock clears all existing image update entries and writes the given
// flags ("stackName/serviceName" → hasUpdate) into BoltDB. Used in mock mode
// to ensure BoltDB state matches mock.yaml on startup and mock reset.
func (s *ImageUpdateStore) SeedFromMock(flags map[string]bool) error {
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

// lastCheckKey is the BoltDB key under the settings bucket that stores the
// Unix timestamp of the last background image update check.
var lastCheckKey = []byte("imageUpdateLastCheck")

// GetLastCheckTime returns the time of the last background image update check.
// Returns zero time if never checked.
func (s *ImageUpdateStore) GetLastCheckTime() (time.Time, error) {
	var t time.Time
	err := s.db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(db.BucketSettings).Get(lastCheckKey)
		if v == nil {
			return nil
		}
		unix, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return nil // treat corrupt value as never checked
		}
		t = time.Unix(unix, 0)
		return nil
	})
	return t, err
}

// SetLastCheckTime records the current time as the last background image update check.
func (s *ImageUpdateStore) SetLastCheckTime(t time.Time) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(db.BucketSettings).Put(lastCheckKey, []byte(strconv.FormatInt(t.Unix(), 10)))
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
