package handlers

import (
	"testing"
	"time"
)

func TestLoginRateLimiterAllow(t *testing.T) {
	t.Parallel()

	rl := NewLoginRateLimiter(3, time.Minute)

	// First 3 attempts should be allowed
	for i := 0; i < 3; i++ {
		if !rl.Allow("user1") {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
	}

	// 4th attempt should be blocked
	if rl.Allow("user1") {
		t.Error("4th attempt should be rate-limited")
	}

	// Different key should still be allowed
	if !rl.Allow("user2") {
		t.Error("different user should not be rate-limited")
	}
}

func TestLoginRateLimiterReset(t *testing.T) {
	t.Parallel()

	rl := NewLoginRateLimiter(2, time.Minute)

	rl.Allow("user1")
	rl.Allow("user1")
	if rl.Allow("user1") {
		t.Error("should be rate-limited after 2 attempts")
	}

	// Reset should clear the limit
	rl.Reset("user1")
	if !rl.Allow("user1") {
		t.Error("should be allowed after reset")
	}
}

func TestLoginRateLimiterWindowExpiry(t *testing.T) {
	t.Parallel()

	// Use a very short window
	rl := NewLoginRateLimiter(2, 50*time.Millisecond)

	rl.Allow("user1")
	rl.Allow("user1")
	if rl.Allow("user1") {
		t.Error("should be rate-limited")
	}

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	if !rl.Allow("user1") {
		t.Error("should be allowed after window expires")
	}
}

func TestLoginRateLimiterCleanup(t *testing.T) {
	t.Parallel()

	rl := NewLoginRateLimiter(5, 50*time.Millisecond)

	rl.Allow("user1")
	rl.Allow("user2")

	time.Sleep(60 * time.Millisecond)
	rl.cleanup()

	rl.mu.Lock()
	remaining := len(rl.attempts)
	rl.mu.Unlock()

	if remaining != 0 {
		t.Errorf("expected 0 entries after cleanup, got %d", remaining)
	}
}

func TestLoginRateLimiterNil(t *testing.T) {
	t.Parallel()

	// Verify nil check pattern works (used in auth.go)
	var rl *LoginRateLimiter
	if rl != nil && !rl.Allow("test") {
		t.Error("nil limiter should not block")
	}
}
