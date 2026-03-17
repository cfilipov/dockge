# Backend Issues Tracker

Identified during code review on 2026-03-16. Each issue is fixed in its own phase with a dedicated commit.

## Status Key
- [ ] Not started
- [~] In progress
- [x] Fixed (commit hash noted)

---

## High Priority

### 1. Unsanitized stack names in shell commands
**Risk:** Stack names from user input flow into `exec.Command` and `rm -rf` without validation. Path traversal (`../`) or shell metacharacters could be dangerous.
**Files:** `internal/handlers/stack.go`, `internal/stack/list.go`, `internal/stack/stack.go`
**Fix:** Add a `ValidateStackName` function that rejects names with path separators, dots-only, empty strings, and non-filesystem-safe characters. Call it at every entry point that accepts a stack name.
**Status:** [ ]

### 2. Unbounded goroutine spawning on WebSocket dispatch
**Risk:** Every incoming message spawns a goroutine (`go s.Dispatch`). Binary frames also spawn per-frame. No backpressure — under load this exhausts memory.
**Files:** `internal/ws/server.go`, `internal/ws/conn.go`
**Fix:** Add a bounded worker pool or semaphore to limit concurrent dispatch goroutines.
**Status:** [ ]

### 3. No per-stack locking
**Risk:** Concurrent saves to the same stack race — last write wins, potential data loss.
**Files:** `internal/handlers/stack.go`
**Fix:** Add a per-stack named mutex (e.g., `sync.Map` of `*sync.Mutex` keyed by stack name) acquired before any read-modify-write operation.
**Status:** [ ]

### 4. Blocking Docker CLI calls in handlers
**Risk:** `exec.Command("docker", "compose", "up")` blocks the handler goroutine. A slow pull blocks WebSocket dispatch for that connection.
**Files:** `internal/handlers/stack.go`
**Fix:** Already partially mitigated by terminal streaming — verify all compose commands run through the terminal manager (async with output streaming), not inline in the handler.
**Status:** [ ]

---

## Medium Priority

### 5. Single broadcast dispatch worker bottleneck
**Risk:** All Docker events serialize through one goroutine. Under heavy container churn, dispatch can't keep up.
**Files:** `internal/handlers/broadcast.go`
**Fix:** Allow multiple dispatch workers or batch events within a time window before querying Docker.
**Status:** [ ]

### 6. Inconsistent error handling — silent failures
**Risk:** Many handlers swallow errors and return empty data. Client can't distinguish "no data" from "error."
**Files:** `internal/handlers/*.go`
**Fix:** Audit all handlers — send error acks with meaningful messages instead of silently returning empty responses.
**Status:** [ ]

### 7. Resource leaks (terminals, stats, event stream)
**Risk:** Completed terminals stay in manager. Stats subscriptions leak on lost unsubscribe. Docker event stream stalls silently.
**Files:** `internal/terminal/manager.go`, `internal/handlers/docker.go`, `internal/handlers/broadcast.go`
**Fix:** Add TTL-based cleanup for completed terminals. Add context timeouts to stats subscriptions. Add keepalive/reconnect to event stream.
**Status:** [x]

### 8. Settings cache race condition
**Risk:** Cache read → miss → DB read → cache write is not atomic. Concurrent reads can see stale data.
**Files:** `internal/models/setting.go`
**Fix:** Use `sync.Map` or add a mutex around the read-miss-write sequence.
**Status:** [x]

---

## Lower Priority

### 9. Brittle YAML parser
**Risk:** Assumes exact indentation. Doesn't strip inline comments. Case-sensitive label matching.
**Files:** `internal/compose/parse.go`
**Fix:** Strip inline comments from image values. Document indentation assumptions (acceptable trade-off for performance vs full YAML parsing).
**Status:** [x]

### 10. Image update checker robustness
**Risk:** No per-image timeout. No remote digest caching. Silent failure on unreachable registry.
**Files:** `internal/handlers/service.go`
**Fix:** Add per-image context timeout. Return check status (success/failed/unknown) alongside update result.
**Status:** [x]

### 11. Auth hardening
**Risk:** No login rate limiting. No password complexity. Password change doesn't invalidate JWTs.
**Files:** `internal/handlers/auth.go`, `internal/models/user.go`
**Fix:** Add rate limiter on login endpoint. Invalidate tokens on password change by rotating the password hash claim.
**Status:** [x]

### 12. CORS InsecureSkipVerify hardcoded
**Risk:** `InsecureSkipVerify: true` in WebSocket accept options. Fine for dev, risky if deployed without reverse proxy.
**Files:** `internal/ws/server.go`
**Fix:** Make origin check configurable; default to same-origin in non-dev mode.
**Status:** [x]

---

## Completed Fixes

_(Entries move here after merge with commit hash and date)_
