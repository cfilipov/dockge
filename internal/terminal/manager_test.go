package terminal

import (
    "strings"
    "sync"
    "testing"
    "time"
)

func TestManagerCreateGet(t *testing.T) {
    t.Parallel()

    m := NewManager()

    // Get nonexistent returns nil
    if m.Get("nope") != nil {
        t.Error("expected nil for nonexistent terminal")
    }

    // Create
    term := m.Create("test", TypePipe)
    if term == nil {
        t.Fatal("expected non-nil terminal")
    }
    if term.Name != "test" {
        t.Errorf("Name = %q", term.Name)
    }
    if term.Type != TypePipe {
        t.Error("expected TypePipe")
    }

    // Get returns the created terminal
    got := m.Get("test")
    if got != term {
        t.Error("Get should return the same terminal")
    }
}

func TestManagerGetOrCreate(t *testing.T) {
    t.Parallel()

    m := NewManager()

    // First call creates
    t1 := m.GetOrCreate("term")
    if t1 == nil {
        t.Fatal("expected non-nil terminal")
    }

    // Second call returns the same one (idempotent)
    t2 := m.GetOrCreate("term")
    if t1 != t2 {
        t.Error("GetOrCreate should be idempotent")
    }
}

func TestManagerRecreateCarriesWriters(t *testing.T) {
    t.Parallel()

    m := NewManager()
    old := m.Create("term", TypePipe)

    var received []string
    old.AddWriter("client1", func(data string) {
        received = append(received, data)
    })

    // Recreate — writers should be carried over
    newTerm := m.Recreate("term", TypePipe)
    if newTerm == old {
        t.Error("Recreate should create a new terminal")
    }

    // Write to new terminal — writer from old should receive it
    newTerm.Write([]byte("hello"))

    if len(received) == 0 {
        t.Error("writer from old terminal should receive data on new terminal")
    }
    if !strings.Contains(received[0], "hello") {
        t.Errorf("expected 'hello', got %q", received[0])
    }

    // Old terminal should be closed
    if !old.closed {
        t.Error("old terminal should be closed after Recreate")
    }
}

func TestManagerRemove(t *testing.T) {
    t.Parallel()

    m := NewManager()
    term := m.Create("rm-test", TypePipe)
    m.Remove("rm-test")

    if m.Get("rm-test") != nil {
        t.Error("terminal should be removed")
    }
    if !term.closed {
        t.Error("removed terminal should be closed")
    }
}

func TestManagerRemoveWriterFromAll(t *testing.T) {
    t.Parallel()

    m := NewManager()
    t1 := m.Create("a", TypePipe)
    t2 := m.Create("b", TypePipe)

    t1.AddWriter("client1", func(string) {})
    t2.AddWriter("client1", func(string) {})

    if t1.WriterCount() != 1 || t2.WriterCount() != 1 {
        t.Fatal("expected 1 writer each")
    }

    m.RemoveWriterFromAll("client1")

    if t1.WriterCount() != 0 {
        t.Error("expected 0 writers on t1")
    }
    if t2.WriterCount() != 0 {
        t.Error("expected 0 writers on t2")
    }
}

func TestRemoveWriterFromAllCleansPipeWithCancel(t *testing.T) {
	t.Parallel()

	m := NewManager()
	term := m.Create("logs--mystack", TypePipe)
	cancelCh := make(chan struct{}, 1)
	term.SetCancel(func() { cancelCh <- struct{}{} })
	term.AddWriter("client1", func(string) {})

	m.RemoveWriterFromAll("client1")

	// Terminal should be removed from manager
	if m.Get("logs--mystack") != nil {
		t.Error("pipe terminal with cancel and zero writers should be removed")
	}
	// Close runs in a goroutine — wait for cancel via channel
	select {
	case <-cancelCh:
		// success
	case <-time.After(time.Second):
		t.Error("cancel should have been called")
	}
}

func TestRemoveWriterFromAllKeepsPTY(t *testing.T) {
	t.Parallel()

	m := NewManager()
	term := m.Create("main-terminal", TypePTY)
	term.AddWriter("client1", func(string) {})

	m.RemoveWriterFromAll("client1")

	// PTY terminal should be kept even with zero writers
	if m.Get("main-terminal") == nil {
		t.Error("PTY terminal should be kept after removing last writer")
	}
}

func TestRemoveWriterFromAllKeepsPipeWithoutCancel(t *testing.T) {
	t.Parallel()

	m := NewManager()
	term := m.Create("compose-mystack", TypePipe)
	// No SetCancel — this is a compose action terminal
	term.AddWriter("client1", func(string) {})

	m.RemoveWriterFromAll("client1")

	// Pipe terminal without cancel should be kept (compose action terminals)
	if m.Get("compose-mystack") == nil {
		t.Error("pipe terminal without cancel should be kept")
	}
}

func TestRemoveWriterFromAllKeepsTerminalWithRemainingWriters(t *testing.T) {
	t.Parallel()

	m := NewManager()
	term := m.Create("logs--mystack", TypePipe)
	term.SetCancel(func() {})
	term.AddWriter("client1", func(string) {})
	term.AddWriter("client2", func(string) {})

	m.RemoveWriterFromAll("client1")

	// Terminal should be kept because client2 is still connected
	if m.Get("logs--mystack") == nil {
		t.Error("terminal with remaining writers should be kept")
	}
	if term.WriterCount() != 1 {
		t.Errorf("expected 1 writer remaining, got %d", term.WriterCount())
	}
}

func TestTerminalWriteBuffer(t *testing.T) {
    t.Parallel()

    term := newTerminal("test", TypePTY) // PTY type: no LF normalization
    term.Write([]byte("hello"))
    term.Write([]byte(" world"))

    buf := term.Buffer()
    if buf != "hello world" {
        t.Errorf("Buffer() = %q", buf)
    }
}

func TestTerminalBufferOverflow(t *testing.T) {
    t.Parallel()

    term := newTerminal("test", TypePTY)

    // Write more than 64KB
    bigData := strings.Repeat("x", 70000)
    term.Write([]byte(bigData))

    buf := term.Buffer()
    if len(buf) > 65536 {
        t.Errorf("buffer should be capped, got len=%d", len(buf))
    }
    // After overflow, keeps last 32KB
    if len(buf) != 32768 {
        t.Errorf("after overflow, buffer should be 32768 bytes, got %d", len(buf))
    }
}

func TestTerminalNormalizeLF(t *testing.T) {
    t.Parallel()

    // Pipe type normalizes \n to \r\n
    pipeTerm := newTerminal("pipe", TypePipe)
    pipeTerm.Write([]byte("line1\nline2\n"))
    buf := pipeTerm.Buffer()
    if !strings.Contains(buf, "\r\n") {
        t.Error("pipe terminal should normalize \\n to \\r\\n")
    }

    // PTY type does not normalize
    ptyTerm := newTerminal("pty", TypePTY)
    ptyTerm.Write([]byte("line1\nline2\n"))
    buf = ptyTerm.Buffer()
    if strings.Contains(buf, "\r\n") {
        t.Error("PTY terminal should not normalize newlines")
    }

    // Already-normalized \r\n should not be doubled
    pipeTerm2 := newTerminal("pipe2", TypePipe)
    pipeTerm2.Write([]byte("line1\r\nline2\r\n"))
    buf = pipeTerm2.Buffer()
    if strings.Contains(buf, "\r\r\n") {
        t.Error("should not double-normalize \\r\\n")
    }
}

func TestTerminalWriterFanOut(t *testing.T) {
    t.Parallel()

    term := newTerminal("test", TypePTY)

    var mu sync.Mutex
    received1 := ""
    received2 := ""

    term.AddWriter("w1", func(data string) {
        mu.Lock()
        received1 += data
        mu.Unlock()
    })
    term.AddWriter("w2", func(data string) {
        mu.Lock()
        received2 += data
        mu.Unlock()
    })

    term.Write([]byte("broadcast"))

    mu.Lock()
    defer mu.Unlock()
    if received1 != "broadcast" {
        t.Errorf("writer1 got %q", received1)
    }
    if received2 != "broadcast" {
        t.Errorf("writer2 got %q", received2)
    }
}

func TestTerminalWriterRemove(t *testing.T) {
    t.Parallel()

    term := newTerminal("test", TypePTY)
    term.AddWriter("w1", func(string) {})

    if term.WriterCount() != 1 {
        t.Fatalf("expected 1 writer, got %d", term.WriterCount())
    }

    term.RemoveWriter("w1")
    if term.WriterCount() != 0 {
        t.Errorf("expected 0 writers after remove, got %d", term.WriterCount())
    }
}

func TestTerminalClose(t *testing.T) {
    t.Parallel()

    cancelCalled := false
    term := newTerminal("test", TypePipe)
    term.SetCancel(func() { cancelCalled = true })

    term.Close()

    if !term.closed {
        t.Error("expected closed=true")
    }
    if !cancelCalled {
        t.Error("expected cancel to be called")
    }

    // Write after close should be no-op
    n, err := term.Write([]byte("data"))
    if n != 0 || err != nil {
        t.Errorf("Write after close: n=%d, err=%v", n, err)
    }
}

func TestTerminalCloseIdempotent(t *testing.T) {
    t.Parallel()

    callCount := 0
    term := newTerminal("test", TypePipe)
    term.SetCancel(func() { callCount++ })

    term.Close()
    term.Close() // should not panic or double-call cancel

    if callCount != 1 {
        t.Errorf("cancel called %d times, want 1", callCount)
    }
}

func TestTerminalAddWriterAfterClose(t *testing.T) {
    t.Parallel()

    term := newTerminal("test", TypePipe)
    term.Close()

    term.AddWriter("late", func(string) {
        t.Error("should never be called")
    })

    // writers should be nil after close, AddWriter should be no-op
    if term.WriterCount() != 0 {
        t.Errorf("expected 0 writers after close, got %d", term.WriterCount())
    }
}

func TestTerminalConcurrentWrite(t *testing.T) {
    t.Parallel()

    term := newTerminal("test", TypePTY)
    var wg sync.WaitGroup

    // 20 goroutines writing concurrently
    for i := 0; i < 20; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := 0; j < 100; j++ {
                term.Write([]byte("data"))
            }
        }()
    }

    wg.Wait()

    // No panic = success. Verify buffer is non-empty.
    if term.Buffer() == "" {
        t.Error("expected non-empty buffer after concurrent writes")
    }
}

func TestTerminalIsRunning(t *testing.T) {
    t.Parallel()

    term := newTerminal("test", TypePipe)
    if term.IsRunning() {
        t.Error("new terminal should not be running")
    }

    term.Close()
    if term.IsRunning() {
        t.Error("closed terminal should not be running")
    }
}

func TestTerminalInputNoOp(t *testing.T) {
    t.Parallel()

    // Pipe terminal with no PTY file — Input should be no-op
    term := newTerminal("test", TypePipe)
    err := term.Input("hello")
    if err != nil {
        t.Errorf("Input on pipe terminal should return nil, got %v", err)
    }
}

func TestTerminalResizeNoOp(t *testing.T) {
    t.Parallel()

    // Pipe terminal with no PTY file — Resize should be no-op
    term := newTerminal("test", TypePipe)
    err := term.Resize(24, 80)
    if err != nil {
        t.Errorf("Resize on pipe terminal should return nil, got %v", err)
    }
}
