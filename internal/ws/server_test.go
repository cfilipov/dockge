package ws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// TestCORSRejectsWrongOriginInProdMode verifies that a WebSocket server
// in production mode (dev=false) rejects connections with a mismatched Origin
// header. The old code had InsecureSkipVerify: true always, which accepted
// all origins. The fix makes it InsecureSkipVerify: s.dev.
func TestCORSRejectsWrongOriginInProdMode(t *testing.T) {
	t.Parallel()

	srv := NewServer(false) // production mode
	// Register a dummy handler so the server is functional
	srv.Handle("ping", func(c *Conn, msg *ClientMessage) {})

	ts := httptest.NewServer(srv)
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt to connect with a wrong Origin header
	_, _, err := websocket.Dial(ctx, "ws"+ts.URL[4:], &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Origin": {"http://evil.com"},
		},
	})
	if err == nil {
		t.Fatal("expected WebSocket dial with wrong Origin to be rejected in prod mode, but it succeeded")
	}

	// Now try with matching origin (derived from the test server URL)
	conn, _, err := websocket.Dial(ctx, "ws"+ts.URL[4:], &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Origin": {ts.URL},
		},
	})
	if err != nil {
		t.Fatalf("expected WebSocket dial with matching Origin to succeed in prod mode: %v", err)
	}
	conn.Close(websocket.StatusNormalClosure, "")
}

// TestCORSAcceptsAnyOriginInDevMode verifies that a WebSocket server
// in dev mode (dev=true) accepts connections from any Origin.
func TestCORSAcceptsAnyOriginInDevMode(t *testing.T) {
	t.Parallel()

	srv := NewServer(true) // dev mode
	srv.Handle("ping", func(c *Conn, msg *ClientMessage) {})

	ts := httptest.NewServer(srv)
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect with a totally different Origin — should succeed in dev mode
	conn, _, err := websocket.Dial(ctx, "ws"+ts.URL[4:], &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Origin": {"http://evil.com"},
		},
	})
	if err != nil {
		t.Fatalf("expected WebSocket dial with any Origin to succeed in dev mode: %v", err)
	}
	conn.Close(websocket.StatusNormalClosure, "")
}
