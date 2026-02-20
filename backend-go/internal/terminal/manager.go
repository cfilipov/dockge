package terminal

import (
    "bytes"
    "os"
    "os/exec"
    "sync"

    "github.com/creack/pty"
)

// TerminalType distinguishes the underlying I/O mechanism.
type TerminalType int

const (
    TypePipe TerminalType = iota // stdout/stderr pipe (compose commands, logs)
    TypePTY                      // pseudo-terminal (interactive shells)
)

// WriteFunc is a callback for streaming terminal output to a WebSocket client.
type WriteFunc func(data string)

// Terminal represents a streaming I/O channel backed by either a pipe or a PTY.
type Terminal struct {
    Name string
    Type TerminalType

    mu      sync.Mutex
    buffer  *bytes.Buffer
    writers map[string]WriteFunc // connID → writer
    closed  bool

    // Process tracking
    cmd    *exec.Cmd
    cancel func() // context cancel or custom cleanup

    // PTY master fd (nil for pipe-based terminals)
    ptyFile *os.File
}

// Manager tracks all active terminals.
type Manager struct {
    mu        sync.RWMutex
    terminals map[string]*Terminal
}

func NewManager() *Manager {
    return &Manager{
        terminals: make(map[string]*Terminal),
    }
}

// Get returns a terminal by name, or nil if not found.
func (m *Manager) Get(name string) *Terminal {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.terminals[name]
}

// GetOrCreate returns an existing terminal or creates a new pipe-based one.
func (m *Manager) GetOrCreate(name string) *Terminal {
    m.mu.Lock()
    defer m.mu.Unlock()

    if t, ok := m.terminals[name]; ok {
        return t
    }
    t := newTerminal(name, TypePipe)
    m.terminals[name] = t
    return t
}

// Create creates a new terminal, replacing any existing one with the same name.
// The old terminal is closed asynchronously.
func (m *Manager) Create(name string, typ TerminalType) *Terminal {
    m.mu.Lock()
    defer m.mu.Unlock()

    if old, ok := m.terminals[name]; ok {
        go old.Close()
    }

    t := newTerminal(name, typ)
    m.terminals[name] = t
    return t
}

// Recreate creates a fresh terminal with a clean buffer, but carries over any
// registered writers from a previous terminal with the same name. This is used
// for compose action terminals where the frontend joins the terminal BEFORE the
// action creates it — without this, the new terminal would have zero subscribers.
func (m *Manager) Recreate(name string, typ TerminalType) *Terminal {
    m.mu.Lock()
    defer m.mu.Unlock()

    var writers map[string]WriteFunc
    if old, ok := m.terminals[name]; ok {
        old.mu.Lock()
        writers = old.writers
        old.writers = make(map[string]WriteFunc) // detach from old terminal
        old.closed = true
        cancelFn := old.cancel
        old.cancel = nil
        old.mu.Unlock()
        // Cancel any running stream (e.g., log tail) on the old terminal
        if cancelFn != nil {
            cancelFn()
        }
    }

    t := newTerminal(name, typ)
    if writers != nil {
        t.writers = writers
    }
    m.terminals[name] = t
    return t
}

// Remove removes and closes a terminal.
func (m *Manager) Remove(name string) {
    m.mu.Lock()
    t, ok := m.terminals[name]
    if ok {
        delete(m.terminals, name)
    }
    m.mu.Unlock()

    if t != nil {
        t.Close()
    }
}

// RemoveWriterFromAll removes a writer (by connID) from every terminal.
// Called when a WebSocket connection disconnects.
func (m *Manager) RemoveWriterFromAll(id string) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    for _, t := range m.terminals {
        t.RemoveWriter(id)
    }
}

func newTerminal(name string, typ TerminalType) *Terminal {
    return &Terminal{
        Name:    name,
        Type:    typ,
        buffer:  &bytes.Buffer{},
        writers: make(map[string]WriteFunc),
    }
}

// Write appends data to the buffer and fans out to all connected writers.
// Implements io.Writer.
//
// For pipe-based terminals, bare \n is normalized to \r\n so that xterm
// renders each line starting at column 0. PTY terminals don't need this
// because the kernel's terminal discipline already handles it.
func (t *Terminal) Write(p []byte) (int, error) {
    t.mu.Lock()
    defer t.mu.Unlock()

    if t.closed {
        return 0, nil
    }

    data := p
    if t.Type == TypePipe {
        data = normalizeLF(p)
    }

    // Buffer output (cap at 64KB, keep last 32KB on overflow)
    t.buffer.Write(data)
    if t.buffer.Len() > 65536 {
        b := t.buffer.Bytes()
        t.buffer.Reset()
        t.buffer.Write(b[len(b)-32768:])
    }

    // Fan out to all connected writers
    s := string(data)
    for _, w := range t.writers {
        w(s)
    }

    return len(p), nil
}

// normalizeLF replaces bare \n (not preceded by \r) with \r\n.
func normalizeLF(p []byte) []byte {
    // Fast path: if no \n at all, return as-is
    if !bytes.Contains(p, []byte{'\n'}) {
        return p
    }
    var buf bytes.Buffer
    buf.Grow(len(p) + 32)
    for i := 0; i < len(p); i++ {
        if p[i] == '\n' && (i == 0 || p[i-1] != '\r') {
            buf.WriteByte('\r')
        }
        buf.WriteByte(p[i])
    }
    return buf.Bytes()
}

// Buffer returns the current terminal buffer content.
func (t *Terminal) Buffer() string {
    t.mu.Lock()
    defer t.mu.Unlock()
    return t.buffer.String()
}

// AddWriter registers a WebSocket client to receive terminal output.
func (t *Terminal) AddWriter(id string, fn WriteFunc) {
    t.mu.Lock()
    defer t.mu.Unlock()
    if !t.closed {
        t.writers[id] = fn
    }
}

// RemoveWriter unregisters a client.
func (t *Terminal) RemoveWriter(id string) {
    t.mu.Lock()
    defer t.mu.Unlock()
    delete(t.writers, id)
}

// WriterCount returns the number of registered writers.
func (t *Terminal) WriterCount() int {
    t.mu.Lock()
    defer t.mu.Unlock()
    return len(t.writers)
}

// Input writes data to the terminal's stdin (PTY master fd).
// For pipe-based terminals this is a no-op.
func (t *Terminal) Input(data string) error {
    t.mu.Lock()
    f := t.ptyFile
    t.mu.Unlock()

    if f != nil {
        _, err := f.WriteString(data)
        return err
    }
    return nil
}

// Resize changes the PTY window size.
// For pipe-based terminals this is a no-op.
func (t *Terminal) Resize(rows, cols uint16) error {
    t.mu.Lock()
    f := t.ptyFile
    t.mu.Unlock()

    if f != nil {
        return pty.Setsize(f, &pty.Winsize{Rows: rows, Cols: cols})
    }
    return nil
}

// IsRunning returns true if the terminal has a running process.
func (t *Terminal) IsRunning() bool {
    t.mu.Lock()
    defer t.mu.Unlock()
    return t.cmd != nil && !t.closed
}

// StartPTY starts a command with a pseudo-terminal. The PTY output is
// continuously read and written to the terminal buffer/fan-out.
func (t *Terminal) StartPTY(cmd *exec.Cmd) error {
    ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
    if err != nil {
        return err
    }

    t.mu.Lock()
    t.cmd = cmd
    t.ptyFile = ptmx
    t.mu.Unlock()

    // Reader goroutine: PTY output → terminal buffer/fan-out
    go func() {
        buf := make([]byte, 4096)
        for {
            n, err := ptmx.Read(buf)
            if n > 0 {
                t.Write(buf[:n])
            }
            if err != nil {
                break
            }
        }
        cmd.Wait()

        t.mu.Lock()
        t.ptyFile = nil
        t.mu.Unlock()
    }()

    return nil
}

// RunPTY starts a command with a pseudo-terminal and blocks until the command
// exits. Output is streamed to the terminal buffer/fan-out in real time.
// Unlike StartPTY, this is synchronous — use it for compose actions where you
// need to know when the command finishes.
func (t *Terminal) RunPTY(cmd *exec.Cmd) error {
    ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
    if err != nil {
        return err
    }

    t.mu.Lock()
    t.cmd = cmd
    t.ptyFile = ptmx
    t.mu.Unlock()

    // Read PTY output until EOF
    buf := make([]byte, 4096)
    for {
        n, readErr := ptmx.Read(buf)
        if n > 0 {
            t.Write(buf[:n])
        }
        if readErr != nil {
            break
        }
    }

    waitErr := cmd.Wait()

    t.mu.Lock()
    t.ptyFile = nil
    t.mu.Unlock()

    return waitErr
}

// SetCancel stores a cancel function called on Close.
// Used for pipe-based terminals with long-running processes (e.g., log streaming).
func (t *Terminal) SetCancel(fn func()) {
    t.mu.Lock()
    defer t.mu.Unlock()
    t.cancel = fn
}

// Close terminates the terminal process and cleans up.
func (t *Terminal) Close() {
    t.mu.Lock()
    defer t.mu.Unlock()

    if t.closed {
        return
    }
    t.closed = true

    if t.cancel != nil {
        t.cancel()
    }
    if t.ptyFile != nil {
        t.ptyFile.Close()
        t.ptyFile = nil
    }
    t.writers = nil
}
