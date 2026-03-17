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

// maxConcurrentDispatch limits how many handler goroutines can run
// simultaneously across all connections. This prevents memory exhaustion
// under load while still allowing plenty of parallelism for a typical
// Dockge deployment.
const maxConcurrentDispatch = 64

// Server manages WebSocket connections and message dispatch.
type Server struct {
    mu    sync.RWMutex
    conns map[*Conn]struct{}

    handlers      map[string]HandlerFunc
    disconnectFn  func(c *Conn)                                  // called when a connection is removed
    binaryHandler func(c *Conn, session *TermSession, data []byte) // handles binary terminal frames

    // dispatchSem bounds concurrent handler goroutines. The read pump
    // blocks on acquire, applying natural backpressure to the client.
    dispatchSem chan struct{}

    // dev controls WebSocket origin checking. When true, all origins are
    // accepted (InsecureSkipVerify). When false, the coder/websocket
    // library enforces same-origin by checking Origin == Host.
    dev bool
}

// NewServer creates a new WebSocket server. The dev parameter controls
// origin checking: true accepts all origins (for development with Vite
// on a different port), false enforces same-origin (production).
func NewServer(dev bool) *Server {
    return &Server{
        conns:       make(map[*Conn]struct{}),
        handlers:    make(map[string]HandlerFunc),
        dispatchSem: make(chan struct{}, maxConcurrentDispatch),
        dev:         dev,
    }
}

// Handle registers a handler for a named event.
func (s *Server) Handle(event string, fn HandlerFunc) {
    s.handlers[event] = fn
}

// OnBinary registers a handler for binary WebSocket frames (terminal data).
// The handler receives the connection, the already-looked-up session, and the
// payload after the 2-byte session ID header.
func (s *Server) OnBinary(fn func(c *Conn, session *TermSession, data []byte)) {
    s.binaryHandler = fn
}

// ServeHTTP upgrades the HTTP request to a WebSocket connection.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
        // In dev mode, accept all origins (Vite runs on a different port).
        // In production, enforce same-origin: the coder/websocket library
        // checks that the Origin header matches the Host header.
        InsecureSkipVerify: s.dev,
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
func Broadcast[T any](s *Server, event string, data T) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    for c := range s.conns {
        SendEvent(c, event, data)
    }
}

// BroadcastAuthenticated sends a push event to all authenticated clients.
func BroadcastAuthenticated[T any](s *Server, event string, data T) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    for c := range s.conns {
        if c.UserID() != 0 {
            SendEvent(c, event, data)
        }
    }
}

// BroadcastAuthenticatedRaw marshals the event payload once and sends the
// pre-encoded bytes to all authenticated connections. For N connections this
// saves (N-1) json.Marshal calls compared to BroadcastAuthenticated.
func BroadcastAuthenticatedRaw[T any](s *Server, event string, data T) {
    payload, err := json.Marshal(ServerMessage[T]{Event: event, Data: data})
    if err != nil {
        slog.Error("ws marshal raw broadcast", "err", err)
        return
    }

    s.mu.RLock()
    defer s.mu.RUnlock()

    for c := range s.conns {
        if c.UserID() != 0 {
            c.writeRaw(payload)
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
    // Acquire a slot from the bounded pool. This blocks the read pump if
    // all slots are in use, applying backpressure to the client rather
    // than spawning unbounded goroutines.
    s.dispatchSem <- struct{}{}
    go func() {
        defer func() { <-s.dispatchSem }()
        s.Dispatch(c, msg)
    }()
}

// Dispatch looks up and invokes the handler for the given message event.
// Exported so the agent handler can re-dispatch unwrapped inner events.
func (s *Server) Dispatch(c *Conn, msg *ClientMessage) {
    h, ok := s.handlers[msg.Event]
    if !ok {
        slog.Warn("ws unknown event", "event", msg.Event)
        if msg.ID != nil {
            SendAck(c, *msg.ID, ErrorResponse{OK: false, Msg: "unknown event: " + msg.Event})
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
