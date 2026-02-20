package ws

import (
    "context"
    "encoding/json"
    "log/slog"
    "sync"
    "time"

    "nhooyr.io/websocket"
)

const (
    writeTimeout = 10 * time.Second
    maxMessageSize = 1 << 20 // 1 MB
)

// Conn wraps a single WebSocket connection.
type Conn struct {
    ws     *websocket.Conn
    server *Server
    userID int // 0 = unauthenticated

    mu      sync.Mutex
    closed  bool
    closeCh chan struct{}
}

func newConn(ws *websocket.Conn, server *Server) *Conn {
    return &Conn{
        ws:      ws,
        server:  server,
        closeCh: make(chan struct{}),
    }
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
func (c *Conn) SendAck(id int64, data interface{}) {
    c.writeJSON(AckMessage{ID: id, Data: data})
}

// SendEvent sends a server push event.
func (c *Conn) SendEvent(event string, args ...interface{}) {
    var a interface{}
    if len(args) == 1 {
        a = args[0]
    } else {
        a = args
    }
    c.writeJSON(ServerMessage{Event: event, Args: a})
}

func (c *Conn) writeJSON(v interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()

    if c.closed {
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
    defer cancel()

    data, err := json.Marshal(v)
    if err != nil {
        slog.Error("ws marshal", "err", err)
        return
    }

    if err := c.ws.Write(ctx, websocket.MessageText, data); err != nil {
        slog.Debug("ws write", "err", err)
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
