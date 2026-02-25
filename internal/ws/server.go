package ws

import (
    "encoding/json"
    "log/slog"
    "net/http"
    "sync"

    "github.com/coder/websocket"
)

// HandlerFunc processes a client message. It receives the connection and the
// raw message. Handlers must return immediately — long-running work should be
// spawned in a goroutine.
type HandlerFunc func(c *Conn, msg *ClientMessage)

// Server manages WebSocket connections and message dispatch.
type Server struct {
    mu    sync.RWMutex
    conns map[*Conn]struct{}

    handlers     map[string]HandlerFunc
    disconnectFn func(c *Conn) // called when a connection is removed
}

func NewServer() *Server {
    return &Server{
        conns:    make(map[*Conn]struct{}),
        handlers: make(map[string]HandlerFunc),
    }
}

// Handle registers a handler for a named event.
func (s *Server) Handle(event string, fn HandlerFunc) {
    s.handlers[event] = fn
}

// ServeHTTP upgrades the HTTP request to a WebSocket connection.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
        // Allow all origins in dev; in production the binary serves the
        // frontend from the same origin so this is fine.
        InsecureSkipVerify: true,
    })
    if err != nil {
        slog.Error("ws accept", "err", err)
        return
    }

    c := newConn(ws, s)
    s.add(c)

    slog.Debug("ws connected", "remote", r.RemoteAddr)

    // Fire the "connect" pseudo-event so handlers can send initial data
    if h, ok := s.handlers["__connect"]; ok {
        h(c, nil)
    }

    // Block on the read pump — this goroutine is owned by net/http
    c.readPump(r.Context())
}

// Broadcast sends a push event to all connected clients.
func (s *Server) Broadcast(event string, args ...interface{}) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    for c := range s.conns {
        c.SendEvent(event, args...)
    }
}

// BroadcastAuthenticated sends a push event to all authenticated clients.
func (s *Server) BroadcastAuthenticated(event string, args ...interface{}) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    for c := range s.conns {
        if c.UserID() != 0 {
            c.SendEvent(event, args...)
        }
    }
}

// BroadcastAuthenticatedRaw marshals the event payload once and sends the
// pre-encoded bytes to all authenticated connections. For N connections this
// saves (N-1) json.Marshal calls compared to BroadcastAuthenticated.
func (s *Server) BroadcastAuthenticatedRaw(event string, args ...interface{}) {
    var a interface{}
    if len(args) == 1 {
        a = args[0]
    } else {
        a = args
    }

    data, err := json.Marshal(ServerMessage{Event: event, Args: a})
    if err != nil {
        slog.Error("ws marshal raw broadcast", "err", err)
        return
    }

    s.mu.RLock()
    defer s.mu.RUnlock()

    for c := range s.conns {
        if c.UserID() != 0 {
            c.writeRaw(data)
        }
    }
}

// BroadcastAuthenticatedBytes sends pre-marshaled JSON bytes to all
// authenticated connections. Use this when you've already serialized the
// ServerMessage and want to avoid re-marshaling.
func (s *Server) BroadcastAuthenticatedBytes(data []byte) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    for c := range s.conns {
        if c.UserID() != 0 {
            c.writeRaw(data)
        }
    }
}

// ConnectionCount returns the number of active connections.
func (s *Server) ConnectionCount() int {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return len(s.conns)
}

// HasAuthenticatedConns returns true if at least one authenticated client
// is connected. This is O(n) in the worst case but short-circuits on the
// first match.
func (s *Server) HasAuthenticatedConns() bool {
    s.mu.RLock()
    defer s.mu.RUnlock()
    for c := range s.conns {
        if c.UserID() != 0 {
            return true
        }
    }
    return false
}

// DisconnectOthers closes all connections except the given one.
func (s *Server) DisconnectOthers(keep *Conn) {
    s.mu.RLock()
    others := make([]*Conn, 0, len(s.conns))
    for c := range s.conns {
        if c != keep {
            others = append(others, c)
        }
    }
    s.mu.RUnlock()

    for _, c := range others {
        c.Close()
    }
}

func (s *Server) add(c *Conn) {
    s.mu.Lock()
    s.conns[c] = struct{}{}
    s.mu.Unlock()
}

func (s *Server) remove(c *Conn) {
    s.mu.Lock()
    delete(s.conns, c)
    s.mu.Unlock()

    if s.disconnectFn != nil {
        s.disconnectFn(c)
    }

    slog.Debug("ws disconnected", "remaining", s.ConnectionCount())
}

// OnDisconnect registers a callback that fires when a connection is removed.
func (s *Server) OnDisconnect(fn func(c *Conn)) {
    s.disconnectFn = fn
}

func (s *Server) dispatch(c *Conn, msg *ClientMessage) {
    // Run each handler in its own goroutine so slow handlers (docker compose ps,
    // docker stats) don't block the read pump and delay other messages.
    go s.Dispatch(c, msg)
}

// Dispatch looks up and invokes the handler for the given message event.
// Exported so the agent handler can re-dispatch unwrapped inner events.
func (s *Server) Dispatch(c *Conn, msg *ClientMessage) {
    h, ok := s.handlers[msg.Event]
    if !ok {
        slog.Warn("ws unknown event", "event", msg.Event)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ErrorResponse{OK: false, Msg: "unknown event: " + msg.Event})
        }
        return
    }
    h(c, msg)
}

// UpgradeHandler returns an http.Handler that upgrades to WebSocket.
// This is a convenience for use with http.ServeMux.
func (s *Server) UpgradeHandler() http.Handler {
    return s
}

// ForEachConn iterates over all connections. The callback must not block.
func (s *Server) ForEachConn(fn func(*Conn)) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    for c := range s.conns {
        fn(c)
    }
}

// HandleConnect registers a handler that fires when a new WebSocket connection
// is established (before the read pump starts).
func (s *Server) HandleConnect(fn func(c *Conn)) {
    s.handlers["__connect"] = func(c *Conn, _ *ClientMessage) {
        fn(c)
    }
}

