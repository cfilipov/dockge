package handlers

import (
	"sync"
	"time"
)

// LoginRateLimiter limits login attempts per key (username or IP).
// Uses a simple sliding window: tracks attempt timestamps and rejects
// if too many attempts occurred within the window.
type LoginRateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	max      int           // max attempts per window
	window   time.Duration // sliding window duration
}

// NewLoginRateLimiter creates a rate limiter that allows `max` attempts
// per `window` duration per key.
func NewLoginRateLimiter(max int, window time.Duration) *LoginRateLimiter {
	return &LoginRateLimiter{
		attempts: make(map[string][]time.Time),
		max:      max,
		window:   window,
	}
}

// Allow checks if a login attempt is allowed for the given key.
// Returns true if under the limit, false if rate-limited.
// Automatically records the attempt if allowed.
func (rl *LoginRateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Prune expired entries
	existing := rl.attempts[key]
	valid := existing[:0]
	for _, t := range existing {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.max {
		rl.attempts[key] = valid
		return false
	}

	rl.attempts[key] = append(valid, now)
	return true
}

// Reset clears rate limit state for a key (e.g., after successful login).
func (rl *LoginRateLimiter) Reset(key string) {
	rl.mu.Lock()
	delete(rl.attempts, key)
	rl.mu.Unlock()
}

// ResetAll clears all rate limit state. Used by dev-mode reset endpoints.
func (rl *LoginRateLimiter) ResetAll() {
	rl.mu.Lock()
	rl.attempts = make(map[string][]time.Time)
	rl.mu.Unlock()
}

// cleanup removes stale entries. Called periodically to prevent memory growth.
func (rl *LoginRateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-rl.window)
	for key, attempts := range rl.attempts {
		valid := attempts[:0]
		for _, t := range attempts {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(rl.attempts, key)
		} else {
			rl.attempts[key] = valid
		}
	}
}

// StartCleanup runs periodic cleanup of expired entries.
func (rl *LoginRateLimiter) StartCleanup(done <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				rl.cleanup()
			}
		}
	}()
}
