package db

import (
    "fmt"
    "log/slog"
    "os"
    "path/filepath"
    "time"

    bolt "go.etcd.io/bbolt"
)

// Bucket names used throughout the application.
var (
    BucketSettings     = []byte("settings")
    BucketUsers        = []byte("users")
    BucketUsersByID    = []byte("users_by_id")
    BucketAgents       = []byte("agents")
    BucketImageUpdates = []byte("image_updates")
)

func Open(dataDir string) (*bolt.DB, error) {
    if err := os.MkdirAll(dataDir, 0755); err != nil {
        return nil, fmt.Errorf("create data dir: %w", err)
    }

    dbPath := filepath.Join(dataDir, "dockge-bolt.db")
    db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
    if err != nil {
        return nil, fmt.Errorf("open bbolt: %w", err)
    }

    // Create all buckets on startup.
    err = db.Update(func(tx *bolt.Tx) error {
        for _, name := range [][]byte{
            BucketSettings,
            BucketUsers,
            BucketUsersByID,
            BucketAgents,
            BucketImageUpdates,
        } {
            if _, err := tx.CreateBucketIfNotExists(name); err != nil {
                return fmt.Errorf("create bucket %s: %w", name, err)
            }
        }
        return nil
    })
    if err != nil {
        db.Close()
        return nil, fmt.Errorf("create buckets: %w", err)
    }

    slog.Info("database ready", "path", dbPath)
    return db, nil
}
