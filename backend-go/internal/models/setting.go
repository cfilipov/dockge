package models

import (
    "database/sql"
    "fmt"
    "log/slog"
    "sync"
    "time"

    "golang.org/x/crypto/bcrypt"
)

const settingCacheTTL = 60 * time.Second

type SettingStore struct {
    db    *sql.DB
    mu    sync.RWMutex
    cache map[string]settingEntry
}

type settingEntry struct {
    value   string
    expires time.Time
}

func NewSettingStore(db *sql.DB) *SettingStore {
    return &SettingStore{
        db:    db,
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
    var value sql.NullString
    err := s.db.QueryRow("SELECT value FROM setting WHERE key = ?", key).Scan(&value)
    if err == sql.ErrNoRows {
        return "", nil
    }
    if err != nil {
        return "", fmt.Errorf("get setting %q: %w", key, err)
    }

    val := ""
    if value.Valid {
        val = value.String
    }

    // Update cache
    s.mu.Lock()
    s.cache[key] = settingEntry{value: val, expires: time.Now().Add(settingCacheTTL)}
    s.mu.Unlock()

    return val, nil
}

// Set stores a setting value (upsert).
func (s *SettingStore) Set(key, value string) error {
    _, err := s.db.Exec(`
        INSERT INTO setting (key, value) VALUES (?, ?)
        ON CONFLICT(key) DO UPDATE SET value = excluded.value
    `, key, value)
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
    rows, err := s.db.Query("SELECT key, value FROM setting")
    if err != nil {
        return nil, fmt.Errorf("get all settings: %w", err)
    }
    defer rows.Close()

    result := make(map[string]string)
    for rows.Next() {
        var key string
        var value sql.NullString
        if err := rows.Scan(&key, &value); err != nil {
            return nil, err
        }
        if value.Valid {
            result[key] = value.String
        }
    }
    return result, rows.Err()
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
