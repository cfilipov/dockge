package ws

import (
    "context"
    "encoding/binary"
    "encoding/json"
    "log/slog"
    "strconv"
    "sync"
    "sync/atomic"
    "time"

    "github.com/coder/websocket"
)

const (
    writeTimeout   = 10 * time.Second
    maxMessageSize = 1 << 20 // 1 MB
)

var connIDCounter uint64

// TermSession holds metadata for a multiplexed terminal session on a connection.
// This is connection-scoped state for fast binary frame dispatch — it doesn't
// import the terminal package. The real terminal logic stays in handlers/.
type TermSession struct {
    TermName    string
    WriterKey   string // connID + ":s" + sessionID
    Interactive bool
}

// Conn wraps a single WebSocket connection.
type Conn struct {
    ws      *websocket.Conn
    server  *Server
    closeCh chan struct{}

    mu     sync.Mutex
    id     string
    userID int // 0 = unauthenticated
    closed bool

    // Terminal session multiplexing
    termMu        sync.RWMutex
    termSessions  map[uint16]*TermSession
    nextSessionID uint16
}

func newConn(ws *websocket.Conn, server *Server) *Conn {
    id := atomic.AddUint64(&connIDCounter, 1)
    return &Conn{
        id:           "c" + strconv.FormatUint(id, 10),
        ws:           ws,
        server:       server,
        closeCh:      make(chan struct{}),
        termSessions: make(map[uint16]*TermSession),
    }
}

// ID returns a unique identifier for this connection.
func (c *Conn) ID() string {
    return c.id
}

// SetUser marks this connection as authenticated.
func (c *Conn) SetUser(userID int) {
    c.mu.Lock()
    c.userID = userID
    c.mu.Unlock()
}

// UserID returns the authenticated user ID (0 if not authenticated).
func (c *Conn) UserID() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.userID
}

// SendAck sends an ack response for a client request.
// Generic to avoid interface boxing — json.Marshal sees the concrete type directly.
func SendAck[T any](c *Conn, id int64, data T) {
    writeJSON(c, AckMessage[T]{ID: id, Data: data})
}

// SendEvent sends a server push event with a single data payload.
// Generic to avoid interface boxing — json.Marshal sees the concrete type directly.
func SendEvent[T any](c *Conn, event string, data T) {
    writeJSON(c, ServerMessage[T]{Event: event, Data: data})
}

func writeJSON[T any](c *Conn, v T) {
    // Marshal outside the lock — this is CPU work, not I/O
    data, err := json.Marshal(v)
    if err != nil {
        slog.Error("ws marshal", "err", err)
        return
    }

    c.writeRaw(data)
}

// writeRaw sends pre-marshalled JSON bytes to the connection.
// Used by BroadcastAuthenticatedRaw to avoid marshalling the same payload per connection.
func (c *Conn) writeRaw(data []byte) {
    c.mu.Lock()
    defer c.mu.Unlock()

    if c.closed {
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
    defer cancel()

    if err := c.ws.Write(ctx, websocket.MessageText, data); err != nil {
        slog.Debug("ws write raw", "err", err)
        c.closeLocked()
    }
}

// WriteBinary sends a binary WebSocket frame.
func (c *Conn) WriteBinary(data []byte) {
    c.mu.Lock()
    defer c.mu.Unlock()

    if c.closed {
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
    defer cancel()

    if err := c.ws.Write(ctx, websocket.MessageBinary, data); err != nil {
        slog.Debug("ws write binary", "err", err)
        c.closeLocked()
    }
}

// AllocSession assigns a session ID and stores the session metadata.
func (c *Conn) AllocSession(s *TermSession) uint16 {
    c.termMu.Lock()
    defer c.termMu.Unlock()

    id := c.nextSessionID
    c.nextSessionID++
    s.WriterKey = c.id + ":s" + strconv.FormatUint(uint64(id), 10)
    c.termSessions[id] = s
    return id
}

// RemoveSession removes and returns a session by ID.
func (c *Conn) RemoveSession(id uint16) *TermSession {
    c.termMu.Lock()
    defer c.termMu.Unlock()

    s := c.termSessions[id]
    delete(c.termSessions, id)
    return s
}

// GetSession returns a session by ID, or nil.
func (c *Conn) GetSession(id uint16) *TermSession {
    c.termMu.RLock()
    defer c.termMu.RUnlock()
    return c.termSessions[id]
}

// DrainSessions atomically removes and returns all sessions.
func (c *Conn) DrainSessions() []*TermSession {
    c.termMu.Lock()
    defer c.termMu.Unlock()

    sessions := make([]*TermSession, 0, len(c.termSessions))
    for _, s := range c.termSessions {
        sessions = append(sessions, s)
    }
    c.termSessions = make(map[uint16]*TermSession)
    return sessions
}

// readPump reads messages from the WebSocket and dispatches them.
func (c *Conn) readPump(ctx context.Context) {
    defer func() {
        c.server.remove(c)
        c.Close()
    }()

    c.ws.SetReadLimit(maxMessageSize)

    for {
        msgType, data, err := c.ws.Read(ctx)
        if err != nil {
            slog.Debug("ws read", "err", err)
            return
        }

        if msgType == websocket.MessageBinary {
            // Binary frame: [2 bytes sessionID BE] [1 byte opcode] [N bytes payload]
            if len(data) < 3 {
                continue
            }
            sessionID := binary.BigEndian.Uint16(data[:2])
            c.termMu.RLock()
            session := c.termSessions[sessionID]
            c.termMu.RUnlock()
            if session == nil {
                continue
            }
            if h := c.server.binaryHandler; h != nil {
                go h(c, session, data[2:])
            }
            continue
        }

        // Text frame: JSON message
        var msg ClientMessage
        if err := json.Unmarshal(data, &msg); err != nil {
            slog.Warn("ws unmarshal", "err", err)
            continue
        }

        c.server.dispatch(c, &msg)
    }
}

// Close shuts down the connection.
func (c *Conn) Close() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.closeLocked()
}

// Done returns a channel that is closed when the connection is closed.
// Handler goroutines can select on this to detect disconnection.
func (c *Conn) Done() <-chan struct{} { return c.closeCh }

func (c *Conn) closeLocked() {
    if c.closed {
        return
    }
    c.closed = true
    close(c.closeCh)
    c.ws.Close(websocket.StatusNormalClosure, "")
}
