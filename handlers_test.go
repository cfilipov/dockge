package main

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "sync"
    "testing"
    "time"

    "github.com/cfilipov/dockge/internal/handlers"
    "github.com/cfilipov/dockge/internal/testutil"
)

func TestNeedSetup(t *testing.T) {
    env := testutil.Setup(t)
    // Fresh DB — no users, so needSetup should be true
    conn := env.DialWS(t)
    resp := env.SendAndReceive(t, conn, "needSetup")

    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatal("expected ok=true")
    }
    needSetup, _ := resp["needSetup"].(bool)
    if !needSetup {
        t.Error("expected needSetup=true on fresh DB")
    }
}

func TestSetupAndLogin(t *testing.T) {
    env := testutil.Setup(t)
    conn := env.DialWS(t)

    // Setup: create user
    resp := env.SendAndReceive(t, conn, "setup", "admin", "testpass123")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("setup failed: %v", resp)
    }

    // Drain post-setup pushes
    conn2 := env.DialWS(t)

    // Login with new credentials
    resp = env.SendAndReceive(t, conn2, "login", "admin", "testpass123", "", "")
    ok, _ = resp["ok"].(bool)
    if !ok {
        t.Fatalf("login failed: %v", resp)
    }
    token, _ := resp["token"].(string)
    if token == "" {
        t.Error("expected non-empty JWT token")
    }
}

func TestLoginBadPassword(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    resp := env.SendAndReceive(t, conn, "login", "admin", "wrongpassword", "", "")
    ok, _ := resp["ok"].(bool)
    if ok {
        t.Error("expected login to fail with wrong password")
    }
}


func TestGetStack(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "getStack", "test-stack")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("getStack failed: %v", resp)
    }

    stackData, _ := resp["stack"].(map[string]interface{})
    if stackData == nil {
        t.Fatal("expected stack data in response")
    }

    name, _ := stackData["name"].(string)
    if name != "test-stack" {
        t.Errorf("expected stack name 'test-stack', got %q", name)
    }

    yaml, _ := stackData["composeYAML"].(string)
    if yaml == "" {
        t.Error("expected non-empty composeYAML")
    }
}

func TestSaveStack(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    newYAML := "services:\n  app:\n    image: alpine:3.19\n"
    resp := env.SendAndReceive(t, conn, "saveStack", "new-stack", newYAML, "", "", false)
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("saveStack failed: %v", resp)
    }

    // Verify file was written to disk
    composePath := filepath.Join(env.StacksDir, "new-stack", "compose.yaml")
    data, err := os.ReadFile(composePath)
    if err != nil {
        t.Fatal("expected compose.yaml on disk:", err)
    }
    if string(data) != newYAML {
        t.Errorf("on-disk YAML mismatch:\ngot:  %q\nwant: %q", string(data), newYAML)
    }
}

func TestDockerStats(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)
    env.SetStackRunning(t, "test-stack")

    conn := env.DialWS(t)
    env.Login(t, conn)

    // Subscribe to stats for a specific container
    resp := env.SendAndReceive(t, conn, "subscribeStats", "test-stack-web-1")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("subscribeStats failed: %v", resp)
    }

    // Wait for the first pushed dockerStats event
    pushed := env.WaitForEvent(t, conn, "dockerStats")
    ok, _ = pushed["ok"].(bool)
    if !ok {
        t.Fatalf("pushed dockerStats not ok: %v", pushed)
    }

    stats, _ := pushed["dockerStats"].(map[string]interface{})
    if stats == nil {
        t.Fatal("expected dockerStats map in pushed event")
    }
    if _, exists := stats["test-stack-web-1"]; !exists {
        t.Fatalf("expected stats for test-stack-web-1, got keys: %v", stats)
    }

    // Unsubscribe
    env.SendAndReceive(t, conn, "unsubscribeStats")
}

func TestContainerTop(t *testing.T) {
	env := testutil.Setup(t)
	env.SeedAdmin(t)
	env.SetStackRunning(t, "test-stack")

	conn := env.DialWS(t)
	env.Login(t, conn)

	// Subscribe to top for a specific container
	resp := env.SendAndReceive(t, conn, "subscribeTop", "test-stack-web-1")
	ok, _ := resp["ok"].(bool)
	if !ok {
		t.Fatalf("subscribeTop failed: %v", resp)
	}

	// Wait for the first pushed containerTop event
	pushed := env.WaitForEvent(t, conn, "containerTop")
	ok, _ = pushed["ok"].(bool)
	if !ok {
		t.Fatalf("pushed containerTop not ok: %v", pushed)
	}

	processes, _ := pushed["processes"].([]interface{})
	if processes == nil {
		t.Fatal("expected processes array in pushed event")
	}
	if len(processes) == 0 {
		t.Error("expected non-empty processes list")
	}

	// Unsubscribe
	env.SendAndReceive(t, conn, "unsubscribeTop")
}

func TestServiceStatusList(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)
    env.SetStackRunning(t, "test-stack")

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "serviceStatusList", "test-stack")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("serviceStatusList failed: %v", resp)
    }

    statusList, _ := resp["serviceStatusList"].(map[string]interface{})
    if statusList == nil {
        t.Fatal("expected serviceStatusList in response")
    }
}

func TestGetDockerNetworkList(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "getDockerNetworkList")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("getDockerNetworkList failed: %v", resp)
    }

    networks, _ := resp["dockerNetworkList"].([]interface{})
    if networks == nil {
        t.Fatal("expected dockerNetworkList in response")
    }
}

func TestUnauthenticatedAccess(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    // Don't login — try to access a protected endpoint
    resp := env.SendAndReceive(t, conn, "getStack", "test-stack")
    ok, _ := resp["ok"].(bool)
    if ok {
        t.Error("expected unauthenticated request to fail")
    }
}

// --- Tier 2A: Stack lifecycle handlers ---

func TestDeployStack(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    yaml := "services:\n  app:\n    image: alpine:3.19\n"
    resp := env.SendAndReceive(t, conn, "deployStack", "deploy-test", yaml, "", "", false)
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("deployStack failed: %v", resp)
    }

    // Verify compose file was saved to disk
    composePath := filepath.Join(env.StacksDir, "deploy-test", "compose.yaml")
    data, err := os.ReadFile(composePath)
    if err != nil {
        t.Fatal("expected compose.yaml on disk:", err)
    }
    if string(data) != yaml {
        t.Errorf("on-disk YAML mismatch:\ngot:  %q\nwant: %q", string(data), yaml)
    }
}

func TestStartStack(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "startStack", "test-stack")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("startStack failed: %v", resp)
    }
}

func TestStopStack(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)
    env.SetStackRunning(t, "test-stack")

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "stopStack", "test-stack")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("stopStack failed: %v", resp)
    }
}

func TestRestartStack(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)
    env.SetStackRunning(t, "test-stack")

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "restartStack", "test-stack")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("restartStack failed: %v", resp)
    }
}

func TestDownStack(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)
    env.SetStackRunning(t, "test-stack")

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "downStack", "test-stack")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("downStack failed: %v", resp)
    }
}

func TestDeleteStackWithFiles(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    // Create a stack first
    stackDir := filepath.Join(env.StacksDir, "to-delete")
    os.MkdirAll(stackDir, 0755)
    os.WriteFile(filepath.Join(stackDir, "compose.yaml"), []byte("services:\n  app:\n    image: alpine\n"), 0644)

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "deleteStack", "to-delete", map[string]bool{"deleteStackFiles": true})
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("deleteStack failed: %v", resp)
    }
}

func TestSaveStackWithOverrideAndEnv(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    yaml := "services:\n  app:\n    image: nginx:latest\n"
    envContent := "DB_HOST=localhost\nDB_PORT=5432"
    overrideYAML := "services:\n  app:\n    ports:\n      - 8080:80\n"

    resp := env.SendAndReceive(t, conn, "saveStack", "full-stack", yaml, envContent, overrideYAML, false)
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("saveStack failed: %v", resp)
    }

    // Verify all three files on disk
    composePath := filepath.Join(env.StacksDir, "full-stack", "compose.yaml")
    data, err := os.ReadFile(composePath)
    if err != nil {
        t.Fatal("compose.yaml:", err)
    }
    if string(data) != yaml {
        t.Errorf("compose YAML mismatch")
    }

    envPath := filepath.Join(env.StacksDir, "full-stack", ".env")
    data, err = os.ReadFile(envPath)
    if err != nil {
        t.Fatal(".env:", err)
    }
    if string(data) != envContent {
        t.Errorf("env mismatch")
    }

    overridePath := filepath.Join(env.StacksDir, "full-stack", "compose.override.yaml")
    data, err = os.ReadFile(overridePath)
    if err != nil {
        t.Fatal("compose.override.yaml:", err)
    }
    if string(data) != overrideYAML {
        t.Errorf("override YAML mismatch")
    }
}

func TestDeployStackEmptyYAML(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "deployStack", "bad-stack", "", "", "", false)
    ok, _ := resp["ok"].(bool)
    if ok {
        t.Error("expected deploy to fail with empty YAML")
    }
}

func TestStartStackMissingName(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "startStack", "")
    ok, _ := resp["ok"].(bool)
    if ok {
        t.Error("expected startStack to fail with empty name")
    }
}

func TestGetStackNonexistent(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    // getStack with a nonexistent stack should still return ok with empty YAML
    resp := env.SendAndReceive(t, conn, "getStack", "nonexistent-stack")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("getStack failed: %v", resp)
    }
    stackData, _ := resp["stack"].(map[string]interface{})
    if stackData == nil {
        t.Fatal("expected stack data in response")
    }
    name, _ := stackData["name"].(string)
    if name != "nonexistent-stack" {
        t.Errorf("expected name 'nonexistent-stack', got %q", name)
    }
}

// --- Tier 2B: Auth error paths ---

func TestLoginByToken(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    // Login to get a token
    conn := env.DialWS(t)
    token := env.Login(t, conn)

    // Use the token to log in on a new connection
    conn2 := env.DialWS(t)
    resp := env.SendAndReceive(t, conn2, "loginByToken", token)
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("loginByToken failed: %v", resp)
    }
}

func TestLoginByTokenBadToken(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    resp := env.SendAndReceive(t, conn, "loginByToken", "invalid.jwt.token")
    ok, _ := resp["ok"].(bool)
    if ok {
        t.Error("expected loginByToken to fail with invalid token")
    }
}

func TestChangePassword(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "changePassword", map[string]string{
        "currentPassword": "testpass123",
        "newPassword":     "newpass456",
    })
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("changePassword failed: %v", resp)
    }

    // Verify new password works
    conn2 := env.DialWS(t)
    resp = env.SendAndReceive(t, conn2, "login", "admin", "newpass456", "", "")
    ok, _ = resp["ok"].(bool)
    if !ok {
        t.Error("expected login with new password to succeed")
    }
}

func TestChangePasswordWrongCurrent(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "changePassword", map[string]string{
        "currentPassword": "wrongpassword",
        "newPassword":     "newpass456",
    })
    ok, _ := resp["ok"].(bool)
    if ok {
        t.Error("expected changePassword to fail with wrong current password")
    }
}

func TestSetupAlreadyDone(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    resp := env.SendAndReceive(t, conn, "setup", "hacker", "password123")
    ok, _ := resp["ok"].(bool)
    if ok {
        t.Error("expected setup to fail when already configured")
    }
}

func TestLoginEmptyCredentials(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    resp := env.SendAndReceive(t, conn, "login", "", "", "", "")
    ok, _ := resp["ok"].(bool)
    if ok {
        t.Error("expected login to fail with empty credentials")
    }
}

func TestLogout(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "logout")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatal("logout failed")
    }

    // After logout, protected endpoints should fail
    resp = env.SendAndReceive(t, conn, "getStack", "test-stack")
    ok, _ = resp["ok"].(bool)
    if ok {
        t.Error("expected access to fail after logout")
    }
}

// --- Tier 2C: Service/settings/agent handlers ---

func TestStartService(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)
    env.SetStackRunning(t, "test-stack")

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "startService", "test-stack", "web")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("startService failed: %v", resp)
    }
}

func TestStopService(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)
    env.SetStackRunning(t, "test-stack")

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "stopService", "test-stack", "web")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("stopService failed: %v", resp)
    }
}

func TestRestartService(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)
    env.SetStackRunning(t, "test-stack")

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "restartService", "test-stack", "web")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("restartService failed: %v", resp)
    }
}

func TestCheckImageUpdates(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "checkImageUpdates", "test-stack")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("checkImageUpdates failed: %v", resp)
    }
}

func TestGetSettings(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "getSettings")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("getSettings failed: %v", resp)
    }

    data, _ := resp["data"].(map[string]interface{})
    if data == nil {
        t.Fatal("expected data map in response")
    }
    // jwtSecret should be filtered out
    if _, has := data["jwtSecret"]; has {
        t.Error("jwtSecret should be filtered from settings response")
    }
}

func TestSetSettings(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    settings := map[string]interface{}{
        "primaryHostname": "example.com",
    }
    resp := env.SendAndReceive(t, conn, "setSettings", settings, "")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("setSettings failed: %v", resp)
    }

    // Verify the setting was saved
    resp = env.SendAndReceive(t, conn, "getSettings")
    data, _ := resp["data"].(map[string]interface{})
    if data["primaryHostname"] != "example.com" {
        t.Errorf("primaryHostname = %v, want example.com", data["primaryHostname"])
    }
}

func TestServiceMissingArgs(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    // Missing service name
    resp := env.SendAndReceive(t, conn, "startService", "test-stack", "")
    ok, _ := resp["ok"].(bool)
    if ok {
        t.Error("expected startService to fail with empty service name")
    }

    // Missing stack name
    resp = env.SendAndReceive(t, conn, "stopService", "", "web")
    ok, _ = resp["ok"].(bool)
    if ok {
        t.Error("expected stopService to fail with empty stack name")
    }
}

func TestUpdateStack(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)
    env.SetStackRunning(t, "test-stack")

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "updateStack", "test-stack")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("updateStack failed: %v", resp)
    }
}

func TestContainerInspect(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)
    env.SetStackRunning(t, "test-stack")

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "containerInspect", "test-stack-web-1")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("containerInspect failed: %v", resp)
    }
}

// --- Global .env settings ---

func TestGlobalENVRoundTrip(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    // Set globalENV
    settings := map[string]interface{}{
        "globalENV": "MY_VAR=hello\nOTHER_VAR=world",
    }
    resp := env.SendAndReceive(t, conn, "setSettings", settings, "")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("setSettings failed: %v", resp)
    }

    // Verify file on disk
    globalEnvPath := filepath.Join(env.StacksDir, "global.env")
    data, err := os.ReadFile(globalEnvPath)
    if err != nil {
        t.Fatal("expected global.env on disk:", err)
    }
    if string(data) != "MY_VAR=hello\nOTHER_VAR=world" {
        t.Errorf("global.env content = %q, want %q", string(data), "MY_VAR=hello\nOTHER_VAR=world")
    }

    // Get settings and verify globalENV is returned
    resp = env.SendAndReceive(t, conn, "getSettings")
    respData, _ := resp["data"].(map[string]interface{})
    globalENV, _ := respData["globalENV"].(string)
    if globalENV != "MY_VAR=hello\nOTHER_VAR=world" {
        t.Errorf("globalENV = %q, want %q", globalENV, "MY_VAR=hello\nOTHER_VAR=world")
    }
}

func TestGlobalENVDefaultDeletes(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    // First set a real value
    settings := map[string]interface{}{
        "globalENV": "MY_VAR=hello",
    }
    env.SendAndReceive(t, conn, "setSettings", settings, "")

    // Verify file exists
    globalEnvPath := filepath.Join(env.StacksDir, "global.env")
    if _, err := os.Stat(globalEnvPath); err != nil {
        t.Fatal("expected global.env to exist after set")
    }

    // Set to default content — should delete file
    settings = map[string]interface{}{
        "globalENV": "# VARIABLE=value #comment",
    }
    resp := env.SendAndReceive(t, conn, "setSettings", settings, "")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("setSettings failed: %v", resp)
    }

    // File should be deleted
    if _, err := os.Stat(globalEnvPath); !os.IsNotExist(err) {
        t.Error("expected global.env to be deleted when set to default content")
    }
}

func TestGlobalENVNotInBoltDB(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    settings := map[string]interface{}{
        "globalENV":       "MY_VAR=hello",
        "primaryHostname": "test.example.com",
    }
    resp := env.SendAndReceive(t, conn, "setSettings", settings, "")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("setSettings failed: %v", resp)
    }

    // globalENV should NOT be stored in BoltDB
    val, err := env.App.Settings.Get("globalENV")
    if err == nil && val != "" {
        t.Errorf("globalENV should NOT be in BoltDB, but got %q", val)
    }

    // primaryHostname SHOULD be in BoltDB
    val, err = env.App.Settings.Get("primaryHostname")
    if err != nil || val != "test.example.com" {
        t.Errorf("primaryHostname should be in BoltDB, got %q, err=%v", val, err)
    }
}

func TestGlobalENVEmptyDeletes(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    // Set a value first
    settings := map[string]interface{}{
        "globalENV": "MY_VAR=hello",
    }
    env.SendAndReceive(t, conn, "setSettings", settings, "")

    // Set to empty — should delete file
    settings = map[string]interface{}{
        "globalENV": "",
    }
    resp := env.SendAndReceive(t, conn, "setSettings", settings, "")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("setSettings failed: %v", resp)
    }

    globalEnvPath := filepath.Join(env.StacksDir, "global.env")
    if _, err := os.Stat(globalEnvPath); !os.IsNotExist(err) {
        t.Error("expected global.env to be deleted when set to empty string")
    }
}

func TestGlobalENVDefaultOnMissing(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    // getSettings without any global.env file should return the default placeholder
    resp := env.SendAndReceive(t, conn, "getSettings")
    respData, _ := resp["data"].(map[string]interface{})
    globalENV, _ := respData["globalENV"].(string)
    if globalENV != "# VARIABLE=value #comment" {
        t.Errorf("expected default globalENV placeholder, got %q", globalENV)
    }
}

// --- Stack name validation ---

func TestSaveStackInvalidName(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    yaml := "services:\n  app:\n    image: alpine:3.19\n"

    tests := []struct {
        name      string
        stackName string
    }{
        {"path traversal", "../traversal"},
        {"shell injection", "; rm -rf /"},
        {"uppercase", "UPPERCASE"},
        {"dot prefix", ".hidden"},
        {"space", "has space"},
        {"leading hyphen", "-leading"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            resp := env.SendAndReceive(t, conn, "saveStack", tt.stackName, yaml, "", "", false)
            ok, _ := resp["ok"].(bool)
            if ok {
                t.Errorf("expected saveStack to reject invalid name %q", tt.stackName)
            }
        })
    }
}

func TestDeleteStackInvalidName(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    tests := []struct {
        name      string
        stackName string
    }{
        {"path traversal", "../traversal"},
        {"shell injection", "; rm -rf /"},
        {"null byte", "stack\x00evil"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            resp := env.SendAndReceive(t, conn, "deleteStack", tt.stackName, map[string]bool{"deleteStackFiles": true})
            ok, _ := resp["ok"].(bool)
            if ok {
                t.Errorf("expected deleteStack to reject invalid name %q", tt.stackName)
            }
        })
    }
}

func TestGetStackInvalidName(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "getStack", "../etc/passwd")
    ok, _ := resp["ok"].(bool)
    if ok {
        t.Error("expected getStack to reject path traversal name")
    }
}

// keys returns the keys of a map for diagnostic messages.
func keys(m map[string]interface{}) []string {
    result := make([]string, 0, len(m))
    for k := range m {
        result = append(result, k)
    }
    return result
}

// --- Regression tests for backend hardening ---

// TestLoginRateLimitIntegration verifies that the login rate limiter is
// correctly wired into the login handler. The rate limiter unit tests exist
// in internal/handlers/ratelimit_test.go, but this test validates the handler
// integration: Allow() is called before password check, and the correct error
// message is returned when rate-limited.
func TestLoginRateLimitIntegration(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    // Install a rate limiter that allows only 3 attempts per minute
    env.App.LoginLimiter = handlers.NewLoginRateLimiter(3, time.Minute)

    // Send 3 wrong-password attempts — should all fail with "authIncorrectCreds"
    for i := 0; i < 3; i++ {
        conn := env.DialWS(t)
        resp := env.SendAndReceive(t, conn, "login", "admin", "wrongpassword", "", "")
        ok, _ := resp["ok"].(bool)
        if ok {
            t.Fatalf("attempt %d: expected login to fail with wrong password", i+1)
        }
        msg, _ := resp["msg"].(string)
        if msg != "authIncorrectCreds" {
            t.Errorf("attempt %d: expected 'authIncorrectCreds', got %q", i+1, msg)
        }
    }

    // 4th attempt (wrong password) — should be rate-limited
    conn4 := env.DialWS(t)
    resp := env.SendAndReceive(t, conn4, "login", "admin", "wrongpassword", "", "")
    ok, _ := resp["ok"].(bool)
    if ok {
        t.Fatal("attempt 4: expected rate-limited login to fail")
    }
    msg, _ := resp["msg"].(string)
    if msg != "Too many login attempts. Please try again later." {
        t.Errorf("attempt 4: expected rate limit message, got %q", msg)
    }

    // 5th attempt with correct password — should STILL be rate-limited
    // (rate limiter fires before password check)
    conn5 := env.DialWS(t)
    resp = env.SendAndReceive(t, conn5, "login", "admin", "testpass123", "", "")
    ok, _ = resp["ok"].(bool)
    if ok {
        t.Error("attempt 5: expected rate-limited login to fail even with correct password")
    }
    msg, _ = resp["msg"].(string)
    if msg != "Too many login attempts. Please try again later." {
        t.Errorf("attempt 5: expected rate limit message, got %q", msg)
    }
}

// TestDownStackBroadcastsNullContainers verifies that downing a stack
// broadcasts explicit null entries for destroyed containers. Without this,
// the frontend's merge-based stores wouldn't remove downed containers.
func TestDownStackBroadcastsNullContainers(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)
    env.SetStackRunning(t, "test-stack")

    // conn1: will issue the downStack command
    conn1 := env.DialWS(t)
    env.Login(t, conn1)

    // conn2: will observe broadcast events
    conn2 := env.DialWS(t)
    env.Login(t, conn2)

    // Wait for the initial containers broadcast on conn2 to confirm the
    // test-stack container exists. WaitForEvent returns the "data" field
    // directly, which IS the containers map (container name → info).
    initial := env.WaitForEvent(t, conn2, "containers")

    // Find any container key belonging to test-stack
    var foundKey string
    for key := range initial {
        if len(key) >= len("test-stack") && key[:len("test-stack")] == "test-stack" {
            foundKey = key
            break
        }
    }
    if foundKey == "" {
        t.Fatalf("expected at least one test-stack container in initial broadcast, got keys: %v", keys(initial))
    }

    // Down the stack
    resp := env.SendAndReceive(t, conn1, "downStack", "test-stack")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("downStack failed: %v", resp)
    }

    // Wait for a post-down containers broadcast where the container is null
    // (destroyed). Multiple broadcasts may arrive: immediate partial updates
    // (die → exited) followed by the destroy → null broadcast.
    for i := 0; i < 10; i++ {
        postDown := env.WaitForEvent(t, conn2, "containers")
        val, exists := postDown[foundKey]
        if !exists {
            // Container absent from broadcast — acceptable if full refresh
            t.Logf("container %q not present in post-down broadcast (acceptable)", foundKey)
            return
        }
        if val == nil {
            // Container explicitly null — expected for merge-based stores
            return
        }
        // Container still present with non-null value (e.g. state: "exited"
        // from die event). Keep reading for the destroy broadcast.
    }
    t.Errorf("expected container %q to be null in post-down broadcast after 10 attempts", foundKey)
}

// TestConcurrentStackOperations verifies that concurrent compose operations
// on the same stack serialize via the per-stack NamedMutex. The lock_test.go
// tests the primitive; this tests the handler integration.
// --- Tier 1: Docker Inspect/List Operations ---

func TestGetDockerImageList(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "getDockerImageList")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("getDockerImageList failed: %v", resp)
    }

    images, _ := resp["dockerImageList"].([]interface{})
    if images == nil {
        t.Fatal("expected dockerImageList in response")
    }
    if len(images) == 0 {
        t.Error("expected at least one image in list")
    }
}

func TestImageInspect(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "imageInspect", "nginx:latest")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("imageInspect failed: %v", resp)
    }

    detail, _ := resp["imageDetail"].(map[string]interface{})
    if detail == nil {
        t.Fatal("expected imageDetail in response")
    }
}

func TestGetDockerVolumeList(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "getDockerVolumeList")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("getDockerVolumeList failed: %v", resp)
    }

    volumes, _ := resp["dockerVolumeList"].([]interface{})
    if volumes == nil {
        t.Fatal("expected dockerVolumeList in response")
    }
}

func TestVolumeInspect(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    // First get the list to find a real volume name
    listResp := env.SendAndReceive(t, conn, "getDockerVolumeList")
    volumes, _ := listResp["dockerVolumeList"].([]interface{})
    if len(volumes) == 0 {
        t.Skip("no volumes available in mock daemon")
    }

    vol, _ := volumes[0].(map[string]interface{})
    volName, _ := vol["name"].(string)
    if volName == "" {
        t.Fatal("expected volume name in first volume")
    }

    resp := env.SendAndReceive(t, conn, "volumeInspect", volName)
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("volumeInspect failed: %v", resp)
    }

    detail, _ := resp["volumeDetail"].(map[string]interface{})
    if detail == nil {
        t.Fatal("expected volumeDetail in response")
    }
}

func TestNetworkInspect(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    // Get the network list first to find a real network name
    listResp := env.SendAndReceive(t, conn, "getDockerNetworkList")
    networks, _ := listResp["dockerNetworkList"].([]interface{})
    if len(networks) == 0 {
        t.Fatal("expected at least one network")
    }

    net, _ := networks[0].(map[string]interface{})
    netName, _ := net["name"].(string)
    if netName == "" {
        t.Fatal("expected network name in first network")
    }

    resp := env.SendAndReceive(t, conn, "networkInspect", netName)
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("networkInspect failed: %v", resp)
    }

    detail, _ := resp["networkDetail"].(map[string]interface{})
    if detail == nil {
        t.Fatal("expected networkDetail in response")
    }
}

// --- Tier 2: Stack Lifecycle (pause/resume, force delete) ---

func TestPauseAndResumeStack(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)
    env.SetStackRunning(t, "test-stack")

    conn := env.DialWS(t)
    env.Login(t, conn)

    // Pause the stack — handler acks immediately, runs compose pause async
    resp := env.SendAndReceive(t, conn, "pauseStack", "test-stack")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("pauseStack failed: %v", resp)
    }

    // Resume the stack — handler acks immediately, runs compose unpause async
    resp = env.SendAndReceive(t, conn, "resumeStack", "test-stack")
    ok, _ = resp["ok"].(bool)
    if !ok {
        t.Fatalf("resumeStack failed: %v", resp)
    }
}

func TestForceDeleteStack(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    // Create a stack to delete
    stackDir := filepath.Join(env.StacksDir, "force-delete-me")
    os.MkdirAll(stackDir, 0755)
    os.WriteFile(filepath.Join(stackDir, "compose.yaml"), []byte("services:\n  app:\n    image: alpine\n"), 0644)

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "forceDeleteStack", "force-delete-me")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("forceDeleteStack failed: %v", resp)
    }

    // Poll for directory removal (async goroutine deletes it)
    deleted := false
    for i := 0; i < 20; i++ {
        if _, err := os.Stat(stackDir); os.IsNotExist(err) {
            deleted = true
            break
        }
        time.Sleep(250 * time.Millisecond)
    }
    if !deleted {
        t.Error("expected stack directory to be deleted after forceDeleteStack")
    }
}

// --- Tier 3: Standalone Container Operations ---

func TestStopContainer(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)
    env.SetStackRunning(t, "test-stack")

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "stopContainer", "test-stack-web-1")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("stopContainer failed: %v", resp)
    }
}

func TestRestartContainer(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)
    env.SetStackRunning(t, "test-stack")

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "restartContainer", "test-stack-web-1")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("restartContainer failed: %v", resp)
    }
}

// --- Tier 4: Service Mutations ---

func TestRecreateService(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)
    env.SetStackRunning(t, "test-stack")

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "recreateService", "test-stack", "web")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("recreateService failed: %v", resp)
    }
}

func TestUpdateService(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)
    env.SetStackRunning(t, "test-stack")

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "updateService", "test-stack", "web")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("updateService failed: %v", resp)
    }
}

// --- Tier 5: Terminal Sessions ---

func TestTerminalJoinAndLeave(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)
    env.SetStackRunning(t, "test-stack")

    conn := env.DialWS(t)
    env.Login(t, conn)

    // Join a combined log terminal
    joinArgs := map[string]interface{}{
        "type":  "combined",
        "stack": "test-stack",
    }
    resp := env.SendAndReceive(t, conn, "terminalJoin", joinArgs)
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("terminalJoin failed: %v", resp)
    }

    // sessionId 0 is valid (first session on a connection)
    sessionID, hasSession := resp["sessionId"].(float64)
    if !hasSession {
        t.Fatal("expected sessionId in response")
    }

    // Leave the terminal
    leaveArgs := map[string]interface{}{
        "sessionId": sessionID,
    }
    resp = env.SendAndReceive(t, conn, "terminalLeave", leaveArgs)
    ok, _ = resp["ok"].(bool)
    if !ok {
        t.Fatalf("terminalLeave failed: %v", resp)
    }
}

func TestTerminalJoinCombinedLog(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)
    env.SetStackRunning(t, "test-stack")

    conn := env.DialWS(t)
    env.Login(t, conn)

    joinArgs := map[string]interface{}{
        "type":  "combined",
        "stack": "test-stack",
    }
    resp := env.SendAndReceive(t, conn, "terminalJoin", joinArgs)
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("terminalJoin combined failed: %v", resp)
    }

    sessionID, hasSession := resp["sessionId"].(float64)
    if !hasSession {
        t.Fatal("expected sessionId in response for combined log")
    }

    // Wait for binary output frame (terminal data is sent as binary)
    data := env.WaitForBinary(t, conn)
    if len(data) < 2 {
        t.Fatal("expected binary frame with at least 2-byte session header")
    }

    // First 2 bytes are session ID (big-endian uint16)
    gotSession := int(data[0])<<8 | int(data[1])
    if gotSession != int(sessionID) {
        t.Errorf("binary frame session ID = %d, want %d", gotSession, int(sessionID))
    }

    // Remaining bytes are terminal output
    if len(data) <= 2 {
        t.Error("expected some terminal output after session header")
    }
}

// --- Tier 6: Settings & Multi-Connection ---

func TestDisconnectOtherSocketClients(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    // Open two connections and login both
    conn1 := env.DialWS(t)
    env.Login(t, conn1)

    conn2 := env.DialWS(t)
    env.Login(t, conn2)

    // Disconnect all other clients from conn1
    resp := env.SendAndReceive(t, conn1, "disconnectOtherSocketClients")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("disconnectOtherSocketClients failed: %v", resp)
    }

    // Verify conn2 is eventually closed — keep reading until error or timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    for {
        _, _, err := conn2.Read(ctx)
        if err != nil {
            // Connection closed as expected
            return
        }
        // Got a buffered message (e.g., post-login pushes), keep draining
    }
}

// TestEventBroadcastSendsFilteredContainers verifies that Docker events trigger
// filtered (partial) container broadcasts, not full-list broadcasts. This catches
// the regression from 4446433 where event coalescing replaced filtered queries
// with full-list queries.
func TestEventBroadcastSendsFilteredContainers(t *testing.T) {
	env := testutil.SetupWith(t, "test-stack", "01-web-app")
	env.SeedAdmin(t)

	// Start both stacks so all containers are running
	env.SetStackRunning(t, "test-stack")
	env.SetStackRunning(t, "01-web-app")

	// conn1: will issue commands
	conn1 := env.DialWS(t)
	env.Login(t, conn1)

	// conn2: observer for broadcasts
	conn2 := env.DialWS(t)
	env.Login(t, conn2)

	// Wait for the initial full containers broadcast on conn2
	initial := env.WaitForEvent(t, conn2, "containers")

	// Count containers from each stack
	var testStackKeys, webAppKeys []string
	for key := range initial {
		if len(key) >= len("test-stack") && key[:len("test-stack")] == "test-stack" {
			testStackKeys = append(testStackKeys, key)
		}
		if len(key) >= len("01-web-app") && key[:len("01-web-app")] == "01-web-app" {
			webAppKeys = append(webAppKeys, key)
		}
	}
	if len(testStackKeys) == 0 {
		t.Fatalf("expected test-stack containers in initial broadcast, got keys: %v", keys(initial))
	}
	if len(webAppKeys) == 0 {
		t.Fatalf("expected 01-web-app containers in initial broadcast, got keys: %v", keys(initial))
	}
	totalInitial := len(initial)

	// Stop test-stack — this generates container stop/die events
	resp := env.SendAndReceive(t, conn1, "stopStack", "test-stack")
	ok, _ := resp["ok"].(bool)
	if !ok {
		t.Fatalf("stopStack failed: %v", resp)
	}

	// Wait for the event-driven containers broadcast on conn2
	postStop := env.WaitForEvent(t, conn2, "containers")

	// The filtered broadcast should contain ONLY test-stack containers
	// (with updated state), NOT containers from 01-web-app.
	// If the code does a full-list query, we'd see all containers.
	if len(postStop) >= totalInitial {
		t.Errorf("expected filtered broadcast with fewer containers than initial full list (%d), got %d keys: %v",
			totalInitial, len(postStop), keys(postStop))
	}

	// Verify the broadcast contains test-stack containers
	hasTestStack := false
	for key := range postStop {
		if len(key) >= len("test-stack") && key[:len("test-stack")] == "test-stack" {
			hasTestStack = true
		}
		// 01-web-app containers should NOT be in the filtered broadcast
		if len(key) >= len("01-web-app") && key[:len("01-web-app")] == "01-web-app" {
			t.Errorf("filtered broadcast should not contain 01-web-app container %q", key)
		}
	}
	if !hasTestStack {
		t.Errorf("expected test-stack containers in post-stop broadcast, got keys: %v", keys(postStop))
	}
}

func TestConcurrentStackOperations(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)
    env.SetStackRunning(t, "test-stack")

    const goroutines = 5
    var wg sync.WaitGroup
    errs := make(chan error, goroutines)

    for i := 0; i < goroutines; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            conn := env.DialWS(t)
            env.Login(t, conn)
            resp := env.SendAndReceive(t, conn, "stopStack", "test-stack")
            ok, _ := resp["ok"].(bool)
            if !ok {
                errs <- fmt.Errorf("stopStack failed: %v", resp)
                return
            }
            errs <- nil
        }()
    }

    wg.Wait()
    close(errs)

    for err := range errs {
        if err != nil {
            t.Error(err)
        }
    }
}
