package ws

import "encoding/json"

// ClientMessage is sent from the browser to the server.
// If ID is non-nil, the client expects an ack response with the same ID.
type ClientMessage struct {
    ID    *int64          `json:"id,omitempty"`
    Event string          `json:"event"`
    Args  json.RawMessage `json:"args"`
}

// AckMessage is sent from the server to the client in response to a request with an ID.
type AckMessage[T any] struct {
    ID   int64 `json:"id"`
    Data T     `json:"data"`
}

// ServerMessage is a server-initiated push (no ack expected).
// Data holds the event payload (a single value of any type).
type ServerMessage[T any] struct {
    Event string `json:"event"`
    Data  T      `json:"data"`
}

// OkResponse is the standard ack payload for successful operations.
type OkResponse struct {
    OK    bool   `json:"ok"`
    Msg   string `json:"msg,omitempty"`
    Token string `json:"token,omitempty"`
}

// ErrorResponse is the standard ack payload for failed operations.
type ErrorResponse struct {
    OK      bool   `json:"ok"`
    Msg     string `json:"msg"`
    MsgI18n bool   `json:"msgi18n,omitempty"`
}

// TerminalJoinArgs is the payload for "terminalJoin" events.
type TerminalJoinArgs struct {
    Type      string `json:"type"`
    Stack     string `json:"stack,omitempty"`
    Service   string `json:"service,omitempty"`
    Container string `json:"container,omitempty"`
    Shell     string `json:"shell,omitempty"`
}

// TerminalJoinResponse is the ack payload for "terminalJoin".
type TerminalJoinResponse struct {
    OK        bool   `json:"ok"`
    SessionID uint16 `json:"sessionId"`
    Msg       string `json:"msg,omitempty"`
}

// TerminalLeaveArgs is the payload for "terminalLeave" events.
type TerminalLeaveArgs struct {
    SessionID uint16 `json:"sessionId"`
}

// TerminalExitedData is the payload for "terminalExited" server push events.
type TerminalExitedData struct {
    SessionID uint16 `json:"sessionId"`
}
