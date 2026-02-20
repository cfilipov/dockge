package models

import "database/sql"

// ImageUpdateStore reads the image_update_cache table.
type ImageUpdateStore struct {
    db *sql.DB
}

func NewImageUpdateStore(db *sql.DB) *ImageUpdateStore {
    return &ImageUpdateStore{db: db}
}

// ImageUpdateEntry represents a cached image update check result.
type ImageUpdateEntry struct {
    StackName   string
    ServiceName string
    HasUpdate   bool
}

// GetAll returns all cached image update entries.
func (s *ImageUpdateStore) GetAll() ([]ImageUpdateEntry, error) {
    rows, err := s.db.Query("SELECT stack_name, service_name, has_update FROM image_update_cache")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var entries []ImageUpdateEntry
    for rows.Next() {
        var e ImageUpdateEntry
        if err := rows.Scan(&e.StackName, &e.ServiceName, &e.HasUpdate); err != nil {
            return nil, err
        }
        entries = append(entries, e)
    }
    return entries, rows.Err()
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

// ServiceUpdatesForStack returns a map of service name → has_update for a given stack.
func (s *ImageUpdateStore) ServiceUpdatesForStack(stackName string) (map[string]bool, error) {
    rows, err := s.db.Query(
        "SELECT service_name, has_update FROM image_update_cache WHERE stack_name = ?",
        stackName,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    result := make(map[string]bool)
    for rows.Next() {
        var svc string
        var hasUpdate bool
        if err := rows.Scan(&svc, &hasUpdate); err != nil {
            return nil, err
        }
        result[svc] = hasUpdate
    }
    return result, rows.Err()
}
