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

    networks, _ := resp["networkList"].([]interface{})
    if networks == nil {
        t.Fatal("expected networkList in response")
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
