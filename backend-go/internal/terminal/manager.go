package terminal

import (
    "bytes"
    "sync"
)

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

// GetOrCreate returns an existing terminal or creates a new one.
func (m *Manager) GetOrCreate(name string) *Terminal {
    m.mu.Lock()
    defer m.mu.Unlock()

    if t, ok := m.terminals[name]; ok {
        return t
    }
    t := &Terminal{
        Name:    name,
        buffer:  &bytes.Buffer{},
        writers: make(map[string]WriteFunc),
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

// WriteFunc is a callback for streaming terminal output to a WebSocket client.
type WriteFunc func(data string)

// Terminal represents a streaming output buffer for a docker compose command.
type Terminal struct {
    Name    string
    buffer  *bytes.Buffer
    mu      sync.RWMutex
    writers map[string]WriteFunc // connID -> writer
    closed  bool
}

// Write appends data to the buffer and fans out to all connected writers.
func (t *Terminal) Write(p []byte) (int, error) {
    t.mu.Lock()
    defer t.mu.Unlock()

    if t.closed {
        return 0, nil
    }

    // Buffer last output (cap at 64KB)
    t.buffer.Write(p)
    if t.buffer.Len() > 65536 {
        // Keep last 32KB
        data := t.buffer.Bytes()
        t.buffer.Reset()
        t.buffer.Write(data[len(data)-32768:])
    }

    // Fan out to all connected writers
    s := string(p)
    for _, w := range t.writers {
        w(s)
    }

    return len(p), nil
}

// Buffer returns the current terminal buffer content.
func (t *Terminal) Buffer() string {
    t.mu.RLock()
    defer t.mu.RUnlock()
    return t.buffer.String()
}

// AddWriter registers a WebSocket client to receive terminal output.
func (t *Terminal) AddWriter(id string, fn WriteFunc) {
    t.mu.Lock()
    defer t.mu.Unlock()
    t.writers[id] = fn
}

// RemoveWriter unregisters a client.
func (t *Terminal) RemoveWriter(id string) {
    t.mu.Lock()
    defer t.mu.Unlock()
    delete(t.writers, id)
}

// Close marks the terminal as closed.
func (t *Terminal) Close() {
    t.mu.Lock()
    defer t.mu.Unlock()
    t.closed = true
    t.writers = nil
}
