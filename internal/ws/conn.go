package ws

import (
    "context"
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

// Conn wraps a single WebSocket connection.
type Conn struct {
    ws      *websocket.Conn
    server  *Server
    closeCh chan struct{}

    mu     sync.Mutex
    id     string
    userID int // 0 = unauthenticated
    closed bool
}

func newConn(ws *websocket.Conn, server *Server) *Conn {
    id := atomic.AddUint64(&connIDCounter, 1)
    return &Conn{
        id:      "c" + strconv.FormatUint(id, 10),
        ws:      ws,
        server:  server,
        closeCh: make(chan struct{}),
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

// readPump reads messages from the WebSocket and dispatches them.
func (c *Conn) readPump(ctx context.Context) {
    defer func() {
        c.server.remove(c)
        c.Close()
    }()

    c.ws.SetReadLimit(maxMessageSize)

    for {
        _, data, err := c.ws.Read(ctx)
        if err != nil {
            slog.Debug("ws read", "err", err)
            return
        }

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

func (c *Conn) closeLocked() {
    if c.closed {
        return
    }
    c.closed = true
    close(c.closeCh)
    c.ws.Close(websocket.StatusNormalClosure, "")
}
