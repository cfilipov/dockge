package models

import (
    "path/filepath"
    "testing"

    "github.com/cfilipov/dockge/internal/db"
)

// openTestDB creates a temp BoltDB for testing.
func openTestDB(t *testing.T) *UserStore {
    t.Helper()
    dir := t.TempDir()
    database, err := db.Open(filepath.Join(dir, "data"))
    if err != nil {
        t.Fatal(err)
    }
    t.Cleanup(func() { database.Close() })
    return NewUserStore(database)
}

func openTestSettingStore(t *testing.T) *SettingStore {
    t.Helper()
    dir := t.TempDir()
    database, err := db.Open(filepath.Join(dir, "data"))
    if err != nil {
        t.Fatal(err)
    }
    t.Cleanup(func() { database.Close() })
    return NewSettingStore(database)
}

func openTestImageUpdateStore(t *testing.T) *ImageUpdateStore {
    t.Helper()
    dir := t.TempDir()
    database, err := db.Open(filepath.Join(dir, "data"))
    if err != nil {
        t.Fatal(err)
    }
    t.Cleanup(func() { database.Close() })
    return NewImageUpdateStore(database)
}

// --- UserStore ---

func TestUserStoreCreateAndFind(t *testing.T) {
    t.Parallel()
    store := openTestDB(t)

    user, err := store.Create("alice", "password123")
    if err != nil {
        t.Fatal(err)
    }
    if user.Username != "alice" {
        t.Errorf("username = %q", user.Username)
    }
    if user.ID == 0 {
        t.Error("expected non-zero ID")
    }

    // Find by username
    found, err := store.FindByUsername("alice")
    if err != nil {
        t.Fatal(err)
    }
    if found == nil {
        t.Fatal("expected user, got nil")
    }
    if found.Username != "alice" {
        t.Errorf("found.Username = %q", found.Username)
    }

    // Find by ID
    foundByID, err := store.FindByID(user.ID)
    if err != nil {
        t.Fatal(err)
    }
    if foundByID == nil {
        t.Fatal("expected user by ID, got nil")
    }
    if foundByID.Username != "alice" {
        t.Errorf("foundByID.Username = %q", foundByID.Username)
    }

    // Find nonexistent
    notFound, err := store.FindByUsername("bob")
    if err != nil {
        t.Fatal(err)
    }
    if notFound != nil {
        t.Error("expected nil for nonexistent user")
    }
}

func TestUserStoreCount(t *testing.T) {
    t.Parallel()
    store := openTestDB(t)

    count, err := store.Count()
    if err != nil {
        t.Fatal(err)
    }
    if count != 0 {
        t.Errorf("initial count = %d, want 0", count)
    }

    store.Create("user1", "pass1")
    store.Create("user2", "pass2")

    count, err = store.Count()
    if err != nil {
        t.Fatal(err)
    }
    if count != 2 {
        t.Errorf("count after 2 creates = %d, want 2", count)
    }
}

func TestUserStoreChangePassword(t *testing.T) {
    t.Parallel()
    store := openTestDB(t)

    user, err := store.Create("admin", "oldpassword")
    if err != nil {
        t.Fatal(err)
    }

    // Verify old password works
    if !VerifyPassword("oldpassword", user.Password) {
        t.Fatal("old password should verify")
    }

    // Change password
    if err := store.ChangePassword(user.ID, "newpassword"); err != nil {
        t.Fatal(err)
    }

    // Verify new password works
    updated, err := store.FindByUsername("admin")
    if err != nil {
        t.Fatal(err)
    }
    if !VerifyPassword("newpassword", updated.Password) {
        t.Error("new password should verify")
    }
    if VerifyPassword("oldpassword", updated.Password) {
        t.Error("old password should no longer verify")
    }
}

// --- SettingStore ---

func TestSettingStoreGetSet(t *testing.T) {
    t.Parallel()
    store := openTestSettingStore(t)

    // Get nonexistent returns empty
    val, err := store.Get("missing")
    if err != nil {
        t.Fatal(err)
    }
    if val != "" {
        t.Errorf("expected empty for missing key, got %q", val)
    }

    // Set and get
    if err := store.Set("hostname", "example.com"); err != nil {
        t.Fatal(err)
    }
    val, err = store.Get("hostname")
    if err != nil {
        t.Fatal(err)
    }
    if val != "example.com" {
        t.Errorf("val = %q, want example.com", val)
    }

    // Overwrite
    if err := store.Set("hostname", "new.example.com"); err != nil {
        t.Fatal(err)
    }
    val, err = store.Get("hostname")
    if err != nil {
        t.Fatal(err)
    }
    if val != "new.example.com" {
        t.Errorf("val = %q, want new.example.com", val)
    }
}

func TestSettingStoreGetAll(t *testing.T) {
    t.Parallel()
    store := openTestSettingStore(t)

    store.Set("key1", "val1")
    store.Set("key2", "val2")

    all, err := store.GetAll()
    if err != nil {
        t.Fatal(err)
    }
    if len(all) != 2 {
        t.Fatalf("expected 2 settings, got %d", len(all))
    }
    if all["key1"] != "val1" {
        t.Errorf("key1 = %q", all["key1"])
    }
}

func TestSettingStoreEnsureJWTSecret(t *testing.T) {
    t.Parallel()
    store := openTestSettingStore(t)

    // First call generates a secret
    secret1, err := store.EnsureJWTSecret()
    if err != nil {
        t.Fatal(err)
    }
    if secret1 == "" {
        t.Fatal("expected non-empty secret")
    }

    // Second call returns the same secret (idempotent)
    secret2, err := store.EnsureJWTSecret()
    if err != nil {
        t.Fatal(err)
    }
    if secret1 != secret2 {
        t.Error("EnsureJWTSecret is not idempotent")
    }
}

func TestSettingStoreInvalidateCache(t *testing.T) {
    t.Parallel()
    store := openTestSettingStore(t)

    store.Set("key", "cached-value")
    store.Get("key") // populate cache

    store.InvalidateCache()

    // Should still work (reads from DB)
    val, err := store.Get("key")
    if err != nil {
        t.Fatal(err)
    }
    if val != "cached-value" {
        t.Errorf("val = %q after cache invalidation", val)
    }
}

// --- ImageUpdateStore ---

func TestImageUpdateStoreUpsertAndQuery(t *testing.T) {
    t.Parallel()
    store := openTestImageUpdateStore(t)

    // Upsert entries
    if err := store.Upsert("stack-a", "web", "nginx:latest", "sha256:aaa", "sha256:bbb", true); err != nil {
        t.Fatal(err)
    }
    if err := store.Upsert("stack-a", "redis", "redis:7", "sha256:ccc", "sha256:ccc", false); err != nil {
        t.Fatal(err)
    }
    if err := store.Upsert("stack-b", "api", "node:20", "sha256:ddd", "sha256:eee", true); err != nil {
        t.Fatal(err)
    }

    // GetAll
    entries, err := store.GetAll()
    if err != nil {
        t.Fatal(err)
    }
    if len(entries) != 3 {
        t.Fatalf("expected 3 entries, got %d", len(entries))
    }

    // StackHasUpdates
    updates, err := store.StackHasUpdates()
    if err != nil {
        t.Fatal(err)
    }
    if !updates["stack-a"] {
        t.Error("stack-a should have updates (web has update)")
    }
    if !updates["stack-b"] {
        t.Error("stack-b should have updates")
    }

    // ServiceUpdatesForStack
    svcUpdates, err := store.ServiceUpdatesForStack("stack-a")
    if err != nil {
        t.Fatal(err)
    }
    if !svcUpdates["web"] {
        t.Error("stack-a/web should have update")
    }
    if svcUpdates["redis"] {
        t.Error("stack-a/redis should not have update")
    }
}

func TestImageUpdateStoreDeleteForStack(t *testing.T) {
    t.Parallel()
    store := openTestImageUpdateStore(t)

    store.Upsert("stack-a", "web", "nginx", "", "", true)
    store.Upsert("stack-a", "redis", "redis", "", "", false)
    store.Upsert("stack-b", "api", "node", "", "", true)

    if err := store.DeleteForStack("stack-a"); err != nil {
        t.Fatal(err)
    }

    entries, _ := store.GetAll()
    if len(entries) != 1 {
        t.Fatalf("expected 1 entry after delete, got %d", len(entries))
    }
    if entries[0].StackName != "stack-b" {
        t.Errorf("remaining entry stack = %q", entries[0].StackName)
    }
}

func TestImageUpdateStoreDeleteService(t *testing.T) {
    t.Parallel()
    store := openTestImageUpdateStore(t)

    store.Upsert("stack-a", "web", "nginx", "", "", true)
    store.Upsert("stack-a", "redis", "redis", "", "", false)

    if err := store.DeleteService("stack-a", "web"); err != nil {
        t.Fatal(err)
    }

    svcUpdates, _ := store.ServiceUpdatesForStack("stack-a")
    if _, ok := svcUpdates["web"]; ok {
        t.Error("web should be deleted")
    }
    if _, ok := svcUpdates["redis"]; !ok {
        t.Error("redis should still exist")
    }
}

func TestImageUpdateStoreUpsertOverwrite(t *testing.T) {
    t.Parallel()
    store := openTestImageUpdateStore(t)

    store.Upsert("stack", "svc", "img", "old", "old", false)
    store.Upsert("stack", "svc", "img", "old", "new", true)

    svcUpdates, _ := store.ServiceUpdatesForStack("stack")
    if !svcUpdates["svc"] {
        t.Error("expected hasUpdate=true after upsert overwrite")
    }

    // Should still be just 1 entry (not 2)
    entries, _ := store.GetAll()
    if len(entries) != 1 {
        t.Errorf("expected 1 entry after upsert, got %d", len(entries))
    }
}
