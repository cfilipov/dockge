package models

import (
    "fmt"
    "log/slog"
    "sync"
    "time"

    bolt "go.etcd.io/bbolt"
    "golang.org/x/crypto/bcrypt"

    "github.com/cfilipov/dockge/internal/db"
)

const settingCacheTTL = 60 * time.Second

type SettingStore struct {
    db    *bolt.DB
    mu    sync.RWMutex
    cache map[string]settingEntry
}

type settingEntry struct {
    value   string
    expires time.Time
}

func NewSettingStore(database *bolt.DB) *SettingStore {
    return &SettingStore{
        db:    database,
        cache: make(map[string]settingEntry),
    }
}

// Get retrieves a setting value by key. Returns "" if not found.
func (s *SettingStore) Get(key string) (string, error) {
    // Check cache
    s.mu.RLock()
    if entry, ok := s.cache[key]; ok && time.Now().Before(entry.expires) {
        s.mu.RUnlock()
        return entry.value, nil
    }
    s.mu.RUnlock()

    // Read from DB
    var val string
    err := s.db.View(func(tx *bolt.Tx) error {
        b := tx.Bucket(db.BucketSettings)
        v := b.Get([]byte(key))
        if v != nil {
            val = string(v)
        }
        return nil
    })
    if err != nil {
        return "", fmt.Errorf("get setting %q: %w", key, err)
    }

    // Update cache
    s.mu.Lock()
    s.cache[key] = settingEntry{value: val, expires: time.Now().Add(settingCacheTTL)}
    s.mu.Unlock()

    return val, nil
}

// Set stores a setting value (upsert).
func (s *SettingStore) Set(key, value string) error {
    err := s.db.Update(func(tx *bolt.Tx) error {
        return tx.Bucket(db.BucketSettings).Put([]byte(key), []byte(value))
    })
    if err != nil {
        return fmt.Errorf("set setting %q: %w", key, err)
    }

    // Update cache
    s.mu.Lock()
    s.cache[key] = settingEntry{value: value, expires: time.Now().Add(settingCacheTTL)}
    s.mu.Unlock()

    return nil
}

// GetAll returns all settings as a map.
func (s *SettingStore) GetAll() (map[string]string, error) {
    result := make(map[string]string)
    err := s.db.View(func(tx *bolt.Tx) error {
        return tx.Bucket(db.BucketSettings).ForEach(func(k, v []byte) error {
            result[string(k)] = string(v)
            return nil
        })
    })
    if err != nil {
        return nil, fmt.Errorf("get all settings: %w", err)
    }
    return result, nil
}

// InvalidateCache clears the settings cache.
func (s *SettingStore) InvalidateCache() {
    s.mu.Lock()
    s.cache = make(map[string]settingEntry)
    s.mu.Unlock()
}

// EnsureJWTSecret creates the JWT secret if it doesn't exist.
// Returns the secret value.
func (s *SettingStore) EnsureJWTSecret() (string, error) {
    secret, err := s.Get("jwtSecret")
    if err != nil {
        return "", err
    }
    if secret != "" {
        return secret, nil
    }

    // Generate new secret: bcrypt(random 64-char string), same as Node.js
    raw, err := GenSecret(secretLength)
    if err != nil {
        return "", fmt.Errorf("generate secret: %w", err)
    }

    hash, err := bcrypt.GenerateFromPassword([]byte(raw), bcryptCost)
    if err != nil {
        return "", fmt.Errorf("hash secret: %w", err)
    }

    secret = string(hash)
    if err := s.Set("jwtSecret", secret); err != nil {
        return "", err
    }

    slog.Info("generated new JWT secret")
    return secret, nil
}
