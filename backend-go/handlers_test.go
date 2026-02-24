package main

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/cfilipov/dockge/backend-go/internal/testutil"
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

func TestRequestStackList(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    resp := env.SendAndReceive(t, conn, "requestStackList")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatal("expected ok=true")
    }
    // The ack is just {ok: true}. The actual stack list is pushed via a
    // separate "agent" event. We verified the handler responds without error.
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

    resp := env.SendAndReceive(t, conn, "dockerStats", "test-stack")
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("dockerStats failed: %v", resp)
    }

    stats, _ := resp["dockerStats"].(map[string]interface{})
    if stats == nil {
        t.Fatal("expected dockerStats map in response")
    }
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

func TestAddAndRemoveAgent(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    // Add agent
    agentData := map[string]string{
        "url":      "https://agent1.example.com",
        "username": "admin",
        "password": "secret",
        "name":     "Test Agent",
    }
    resp := env.SendAndReceive(t, conn, "addAgent", agentData)
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("addAgent failed: %v", resp)
    }

    // Remove agent
    resp = env.SendAndReceive(t, conn, "removeAgent", "https://agent1.example.com")
    ok, _ = resp["ok"].(bool)
    if !ok {
        t.Fatalf("removeAgent failed: %v", resp)
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

func TestUpdateAgent(t *testing.T) {
    env := testutil.Setup(t)
    env.SeedAdmin(t)

    conn := env.DialWS(t)
    env.Login(t, conn)

    // Add agent first
    agentData := map[string]string{
        "url":      "https://agent2.example.com",
        "username": "admin",
        "password": "secret",
        "name":     "Original Name",
    }
    resp := env.SendAndReceive(t, conn, "addAgent", agentData)
    ok, _ := resp["ok"].(bool)
    if !ok {
        t.Fatalf("addAgent failed: %v", resp)
    }

    // Update agent name
    resp = env.SendAndReceive(t, conn, "updateAgent", "https://agent2.example.com", "New Name")
    ok, _ = resp["ok"].(bool)
    if !ok {
        t.Fatalf("updateAgent failed: %v", resp)
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
