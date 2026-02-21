package main

import (
    "os"
    "os/exec"
    "runtime"
    "runtime/debug"
    "runtime/pprof"
    "testing"

    "github.com/cfilipov/dockge/backend-go/internal/docker"
    "github.com/cfilipov/dockge/backend-go/internal/testutil"
)

const maxBinarySizeBytes = 20 * 1024 * 1024 // 20 MB

func TestBinarySize(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping binary size check in short mode")
    }

    tmpFile := t.TempDir() + "/dockge-backend"

    cmd := exec.Command("go", "build",
        "-ldflags=-s -w",
        "-trimpath",
        "-o", tmpFile,
        ".",
    )
    cmd.Dir = "."
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err := cmd.Run(); err != nil {
        t.Fatal("build failed:", err)
    }

    info, err := os.Stat(tmpFile)
    if err != nil {
        t.Fatal(err)
    }

    size := info.Size()
    sizeMB := float64(size) / 1024 / 1024
    t.Logf("binary size: %.2f MB (%d bytes)", sizeMB, size)

    if size > maxBinarySizeBytes {
        t.Errorf("binary size %.2f MB exceeds budget of %d MB", sizeMB, maxBinarySizeBytes/(1024*1024))
    }
}

const maxHeapMB = 50

func TestMemoryBudget(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping memory budget test in short mode")
    }

    env := testutil.Setup(t)
    env.SeedAdmin(t)
    env.SetStackRunning(t, "test-stack")

    conn := env.DialWS(t)
    env.Login(t, conn)

    // Run representative workload
    for i := 0; i < 5; i++ {
        env.SendAndReceive(t, conn, "requestStackList")
    }
    for i := 0; i < 3; i++ {
        env.SendAndReceive(t, conn, "dockerStats", "test-stack")
    }
    for i := 0; i < 3; i++ {
        env.SendAndReceive(t, conn, "serviceStatusList", "test-stack")
    }
    env.SendAndReceive(t, conn, "getStack", "test-stack")
    env.SendAndReceive(t, conn, "getDockerNetworkList")

    runtime.GC()
    debug.FreeOSMemory()
    var ms runtime.MemStats
    runtime.ReadMemStats(&ms)

    heapMB := ms.HeapInuse / 1024 / 1024
    if heapMB > maxHeapMB {
        t.Errorf("heap %d MB exceeds %d MB budget", heapMB, maxHeapMB)
    }
    t.Logf("heap: %d MB, alloc: %d MB, sys: %d MB",
        heapMB, ms.HeapAlloc/1024/1024, ms.Sys/1024/1024)
}

func BenchmarkLogin(b *testing.B) {
    env := testutil.Setup(b)
    env.SeedAdmin(b)

    b.ReportAllocs()
    b.ResetTimer()
    for b.Loop() {
        conn := env.DialWS(b)
        env.SendAndReceive(b, conn, "login", "admin", "testpass123", "", "")
        conn.Close(4000, "bench done")
    }
}

func BenchmarkRequestStackList(b *testing.B) {
    env := testutil.Setup(b)
    env.SeedAdmin(b)

    conn := env.DialWS(b)
    env.Login(b, conn)

    b.ReportAllocs()
    b.ResetTimer()
    for b.Loop() {
        env.SendAndReceive(b, conn, "requestStackList")
    }
}

func BenchmarkGetStack(b *testing.B) {
    env := testutil.Setup(b)
    env.SeedAdmin(b)

    conn := env.DialWS(b)
    env.Login(b, conn)

    b.ReportAllocs()
    b.ResetTimer()
    for b.Loop() {
        env.SendAndReceive(b, conn, "getStack", "test-stack")
    }
}

func BenchmarkServiceStatusList(b *testing.B) {
    env := testutil.Setup(b)
    env.SeedAdmin(b)
    env.SetStackRunning(b, "test-stack")

    conn := env.DialWS(b)
    env.Login(b, conn)

    b.ReportAllocs()
    b.ResetTimer()
    for b.Loop() {
        env.SendAndReceive(b, conn, "serviceStatusList", "test-stack")
    }
}

func BenchmarkDockerStats(b *testing.B) {
    env := testutil.Setup(b)
    env.SeedAdmin(b)
    env.SetStackRunning(b, "test-stack")

    conn := env.DialWS(b)
    env.Login(b, conn)

    b.ReportAllocs()
    b.ResetTimer()
    for b.Loop() {
        env.SendAndReceive(b, conn, "dockerStats", "test-stack")
    }
}

func BenchmarkRequestStackList200(b *testing.B) {
    env := testutil.SetupFull(b)
    env.SeedAdmin(b)

    // Set all stacks to their default states
    state := docker.DefaultDevState()
    for name, status := range state.All() {
        env.State.Set(name, status)
    }

    conn := env.DialWS(b)
    env.Login(b, conn)

    b.ReportAllocs()
    b.ResetTimer()
    for b.Loop() {
        env.SendAndReceive(b, conn, "requestStackList")
    }
}

// BenchmarkRequestStackList200_HeapProfile writes heap profiles before and after
// a workload for offline analysis with `go tool pprof`.
// Usage: go test -bench=BenchmarkRequestStackList200_HeapProfile -benchtime=1x -run='^$' .
// Then:  go tool pprof -http=:8080 heap-after.prof
func BenchmarkRequestStackList200_HeapProfile(b *testing.B) {
    env := testutil.SetupFull(b)
    env.SeedAdmin(b)

    state := docker.DefaultDevState()
    for name, status := range state.All() {
        env.State.Set(name, status)
    }

    conn := env.DialWS(b)
    env.Login(b, conn)

    runtime.GC()
    writeHeapProfile(b, "heap-before.prof")

    b.ResetTimer()
    for i := 0; i < 100; i++ {
        env.SendAndReceive(b, conn, "requestStackList")
    }
    b.StopTimer()

    runtime.GC()
    writeHeapProfile(b, "heap-after.prof")
}

func writeHeapProfile(tb testing.TB, filename string) {
    tb.Helper()
    f, err := os.Create(filename)
    if err != nil {
        tb.Fatalf("create heap profile: %v", err)
    }
    defer f.Close()
    if err := pprof.WriteHeapProfile(f); err != nil {
        tb.Fatalf("write heap profile: %v", err)
    }
    tb.Logf("wrote heap profile to %s", filename)
}
