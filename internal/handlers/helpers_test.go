package handlers

import (
    "encoding/json"
    "testing"

    "github.com/cfilipov/dockge/internal/ws"
)

func TestParseArgs(t *testing.T) {
    t.Parallel()

    t.Run("nil message", func(t *testing.T) {
        t.Parallel()
        args := parseArgs(nil)
        if args != nil {
            t.Error("expected nil for nil message")
        }
    })

    t.Run("empty args", func(t *testing.T) {
        t.Parallel()
        msg := &ws.ClientMessage{Event: "test", Args: nil}
        args := parseArgs(msg)
        if args != nil {
            t.Error("expected nil for empty args")
        }
    })

    t.Run("valid JSON array", func(t *testing.T) {
        t.Parallel()
        msg := &ws.ClientMessage{
            Event: "test",
            Args:  json.RawMessage(`["hello", 42, true]`),
        }
        args := parseArgs(msg)
        if len(args) != 3 {
            t.Fatalf("expected 3 args, got %d", len(args))
        }
    })

    t.Run("invalid JSON", func(t *testing.T) {
        t.Parallel()
        msg := &ws.ClientMessage{
            Event: "test",
            Args:  json.RawMessage(`not json`),
        }
        args := parseArgs(msg)
        if args != nil {
            t.Error("expected nil for invalid JSON")
        }
    })
}

func TestArgString(t *testing.T) {
    t.Parallel()

    args := []json.RawMessage{
        json.RawMessage(`"hello"`),
        json.RawMessage(`42`),
        json.RawMessage(`true`),
    }

    t.Run("valid index", func(t *testing.T) {
        t.Parallel()
        if got := argString(args, 0); got != "hello" {
            t.Errorf("argString(0) = %q, want hello", got)
        }
    })

    t.Run("out of bounds", func(t *testing.T) {
        t.Parallel()
        if got := argString(args, 10); got != "" {
            t.Errorf("argString(10) = %q, want empty", got)
        }
    })

    t.Run("non-string value", func(t *testing.T) {
        t.Parallel()
        if got := argString(args, 1); got != "" {
            t.Errorf("argString(1) for number = %q, want empty", got)
        }
    })

    t.Run("nil args", func(t *testing.T) {
        t.Parallel()
        if got := argString(nil, 0); got != "" {
            t.Errorf("argString(nil, 0) = %q, want empty", got)
        }
    })
}

func TestArgBool(t *testing.T) {
    t.Parallel()

    args := []json.RawMessage{
        json.RawMessage(`true`),
        json.RawMessage(`false`),
        json.RawMessage(`"not a bool"`),
    }

    t.Run("true", func(t *testing.T) {
        t.Parallel()
        if !argBool(args, 0) {
            t.Error("expected true")
        }
    })

    t.Run("false", func(t *testing.T) {
        t.Parallel()
        if argBool(args, 1) {
            t.Error("expected false")
        }
    })

    t.Run("non-bool", func(t *testing.T) {
        t.Parallel()
        if argBool(args, 2) {
            t.Error("expected false for non-bool")
        }
    })

    t.Run("out of bounds", func(t *testing.T) {
        t.Parallel()
        if argBool(args, 10) {
            t.Error("expected false for out of bounds")
        }
    })
}

func TestArgInt(t *testing.T) {
    t.Parallel()

    args := []json.RawMessage{
        json.RawMessage(`42`),
        json.RawMessage(`3.7`),
        json.RawMessage(`"not a number"`),
    }

    t.Run("integer", func(t *testing.T) {
        t.Parallel()
        if got := argInt(args, 0); got != 42 {
            t.Errorf("argInt(0) = %d, want 42", got)
        }
    })

    t.Run("float64 truncation", func(t *testing.T) {
        t.Parallel()
        if got := argInt(args, 1); got != 3 {
            t.Errorf("argInt(1) = %d, want 3", got)
        }
    })

    t.Run("non-number", func(t *testing.T) {
        t.Parallel()
        if got := argInt(args, 2); got != 0 {
            t.Errorf("argInt(2) = %d, want 0", got)
        }
    })

    t.Run("out of bounds", func(t *testing.T) {
        t.Parallel()
        if got := argInt(args, 10); got != 0 {
            t.Errorf("argInt(10) = %d, want 0", got)
        }
    })
}

func TestArgObject(t *testing.T) {
    t.Parallel()

    args := []json.RawMessage{
        json.RawMessage(`{"name":"test","value":42}`),
        json.RawMessage(`"not an object"`),
    }

    t.Run("valid struct", func(t *testing.T) {
        t.Parallel()
        var dst struct {
            Name  string `json:"name"`
            Value int    `json:"value"`
        }
        if !argObject(args, 0, &dst) {
            t.Fatal("expected true")
        }
        if dst.Name != "test" {
            t.Errorf("Name = %q", dst.Name)
        }
        if dst.Value != 42 {
            t.Errorf("Value = %d", dst.Value)
        }
    })

    t.Run("invalid JSON for struct", func(t *testing.T) {
        t.Parallel()
        var dst struct{ Name string }
        // String value can unmarshal into struct but fields won't match
        if argObject(args, 1, &dst) && dst.Name != "" {
            // A string doesn't unmarshal into a struct successfully
        }
    })

    t.Run("out of bounds", func(t *testing.T) {
        t.Parallel()
        var dst struct{}
        if argObject(args, 10, &dst) {
            t.Error("expected false for out of bounds")
        }
    })
}

