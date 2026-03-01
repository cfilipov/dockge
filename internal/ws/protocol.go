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
