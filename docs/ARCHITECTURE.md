# Go Backend Architecture

This document describes the architecture of the Dockge Go backend. It covers the
design philosophy, core data flows, resource management strategies, and the mock
system that enables development without a real Docker daemon.

## Table of Contents

- [Design Philosophy](#design-philosophy)
- [Broadcast Registration Flow](#broadcast-registration-flow)
- [Startup and Idle Cleanup](#startup-and-idle-cleanup)
- [Terminal Lifecycle](#terminal-lifecycle)
- [Docker CLI vs SDK](#docker-cli-vs-sdk)
- [Memory Optimization Techniques](#memory-optimization-techniques)
- [Mock System Architecture](#mock-system-architecture)

---

## Design Philosophy

Dockge is a **low-memory, non-blocking ephemeral broker** between the Docker
daemon and a Vue.js frontend. It is not a monitoring system that accumulates
state — it is a thin relay that queries Docker on demand, formats the response,
and pushes it over a WebSocket.

Three principles guide every design decision:

**1. Respond immediately, enrich asynchronously.** No request handler ever blocks
on a slow Docker API call, registry lookup, or filesystem scan before returning
something to the client. Data that is available right now (even if incomplete) is
sent first. Richer data arrives later as WebSocket pushes. The frontend renders
progressively — each broadcast channel maps to a Pinia store, and Vue's
reactivity updates the UI as stores fill in.

**2. Let the frontend do the work.** The backend sends raw lists (containers,
images, networks, volumes) and the frontend joins them together. For example,
stack status is derived client-side by matching containers to stacks via compose
project labels, not computed on the backend. This keeps the backend stateless and
the broadcast payloads simple.

**3. Conserve resources when nobody is looking.** Dockge is a self-hosted tool
with an intermittent usage pattern — active for a few minutes while managing
stacks, then idle for hours or days. The backend adapts: when no clients are
connected, the Docker Events watcher shuts down, timers stop, and heap pages are
returned to the OS. When a client connects, everything spins back up in
milliseconds.

---

## Broadcast Registration Flow

The broadcast system is the backbone of the UI. Six independent channels deliver
different slices of Docker state to connected clients:

| Channel | Source | Pinia Store |
|-------------|-------------------------------------|-------------|
| `stacks` | Filesystem scan + compose parse | `stackStore` |
| `containers` | `Docker.ContainerListDetailed()` | `containerStore` |
| `networks` | `Docker.NetworkList()` | `networkStore` |
| `images` | `Docker.ImageList()` | `imageStore` |
| `volumes` | `Docker.VolumeList()` | `volumeStore` |
| `updates` | BoltDB `imageUpdates` bucket | `updateStore` |

### Initial hydration

When a client authenticates, `AfterLogin()` (auth.go) fires all six broadcast
channels as independent goroutines — each sends to the connection as soon as its
data is ready, with no channel waiting on any other:

```
AfterLogin(conn)
  ├── EnsureWatcherRunning()         // lazy-start the Docker Events watcher
  ├── go sendToConn(stacks)          // dir scan + YAML parse
  ├── go sendToConn(containers)      // Docker API call
  ├── go sendToConn(networks)        // Docker API call
  ├── go sendToConn(images)          // Docker API call
  ├── go sendToConn(volumes)         // Docker API call
  └── go sendToConn(updates)         // BoltDB read
```

There is no `WaitGroup` or ordering — all six goroutines are fire-and-forget. The
frontend renders progressively as each store populates via its WebSocket channel.

### Ongoing updates

After initial hydration, two event sources drive subsequent broadcasts:

**Docker Events API.** A single persistent `GET /events` connection (via the SDK)
receives real-time container, network, image, and volume events. Each event is
mapped to its corresponding broadcast channel and fed into a **trailing-edge
debouncer** (200ms). Rapid-fire events (e.g., ten containers starting during
`docker compose up`) collapse into a single broadcast 200ms after the storm stops.

```
Docker Events stream
  → consumeBroadcastEvents()
      → EventBus.Publish(evt)              // fan-out to terminal subscribers
      → debouncer.trigger("containers")    // 200ms trailing-edge
          → broadcastContainers()
              → Docker.ContainerListDetailed()
              → FNV-1a hash comparison
              → BroadcastAuthenticatedBytes()  // only if data changed
```

**fsnotify compose file watcher.** `compose.StartWatcher()` monitors the stacks
directory for file changes (create, write, remove, rename). Events are debounced
per-stack (200ms) and trigger a stacks broadcast. This covers cases that Docker
events don't: editing a compose.yaml, adding a new stack directory, or deleting
one.

### Deduplication

Every broadcast passes through `broadcastIfChanged()`, which computes an FNV-1a
hash of the JSON payload and compares it to the last hash sent for that channel.
If unchanged, the broadcast is skipped. This prevents redundant WebSocket writes
when a Docker event fires but the relevant data hasn't actually changed (common
with health check events).

### Pre-marshaled bytes

The JSON payload marshaled for hashing is reused as the wire payload — it goes
directly to `BroadcastAuthenticatedBytes()`, which writes the same `[]byte` to
every authenticated connection. This avoids marshaling once per client.

### Key constants

| Value | Purpose |
|-------|---------|
| 200ms | Debounce interval (trailing edge) for both Docker events and fsnotify |
| 15s | Context timeout on all Docker API calls in broadcast functions |
| 5 retries | Max Docker Events reconnect attempts before `os.Exit(1)` |
| 1s → 30s | Exponential backoff for Events reconnection |

### Key files

- `internal/handlers/broadcast.go` — Broadcast channels, debouncing, FNV dedup, watcher lifecycle
- `internal/handlers/eventbus.go` — Shared Docker event fan-out
- `internal/handlers/auth.go` — `AfterLogin()` trigger + initial hydration goroutines
- `internal/ws/server.go` — WebSocket server, pre-marshaled broadcast delivery
- `internal/compose/watcher.go` — fsnotify compose file watcher

---

## Startup and Idle Cleanup

The server starts with a minimal footprint and scales up only when needed.

### Startup sequence (main.go)

1. Parse config (CLI flags + env vars)
2. Set `GOMAXPROCS` (default: 1)
3. Open BoltDB
4. Create WebSocket server and HTTP mux
5. Initialize model stores (users, settings, image updates)
6. Create Docker SDK client (connects to `DOCKER_HOST`)
7. Create terminal manager
8. Register all WebSocket handlers
9. Call `InitBroadcast()` — creates `broadcastState` and `EventBus` but does **not** start the watcher
10. Start compose file watcher (fsnotify)
11. Start image update checker (background timer, 6h default)
12. Start periodic `FreeOSMemory()` goroutine (1-minute tick)
13. Start HTTP server

**The broadcast watcher is NOT started at boot.** The server sits idle, consuming
minimal resources, until the first client authenticates.

### State transition diagram

```
                    ┌─────────────────────────────────────────────┐
                    │                                             │
                    ▼                                             │
              ┌──────────┐    first client       ┌────────────┐  │
  Server ──►  │   IDLE   │ ──authenticates──►    │   ACTIVE   │  │
  starts      │          │                       │            │  │
              │ no watcher│   EnsureWatcher-     │ watcher +  │  │
              │ no events │   Running()          │ events +   │  │
              │ minimal   │                      │ broadcasts │  │
              │ memory    │                      │            │  │
              └──────────┘                       └─────┬──────┘  │
                    ▲                                  │         │
                    │         last client              │         │
                    │         disconnects              │         │
                    │                                  ▼         │
                    │                          ┌──────────────┐  │
                    │         60s grace        │   DRAINING   │  │
                    │◄───────timer fires──────│              │  │
                    │                          │ 60s timer    │  │
                    │  stopWatcherLocked()     │ running      │  │
                    │  + FreeOSMemory()        │              │  │
                    │                          └──────┬───────┘  │
                    │                                 │          │
                    │            new client           │          │
                    │            connects within 60s  │          │
                    │                                 └──────────┘
                    │                           timer cancelled
```

### Watcher lifecycle details

**`EnsureWatcherRunning()`** — Called on every successful authentication. Under
`watcherMu`:
1. Cancels any pending idle timer (prevents a scheduled stop)
2. If watcher already running, returns immediately (idempotent)
3. Otherwise, creates a child context and calls `StartBroadcastWatcher()`

**`ScheduleWatcherStop()`** — Called from the disconnect handler when
`HasAuthenticatedConns()` returns false. Under `watcherMu`:
1. If watcher not running, returns
2. Sets `idleTimer = time.AfterFunc(60s, ...)`
3. When timer fires: double-checks no clients reconnected, then calls
   `stopWatcherLocked()` + `debug.FreeOSMemory()`

**`stopWatcherLocked()`** — Cancels the watcher context, stops the debouncer,
and resets the FNV hash cache to empty. The hash reset ensures the next watcher
start re-broadcasts everything fresh (no stale hash suppression).

### Disconnect cleanup

When a WebSocket connection drops, the disconnect handler:
1. Calls `terms.RemoveWriterFromAll(conn.ID())` — removes this connection as a
   writer from all terminals, cleaning up orphaned pipe terminals
2. If no authenticated connections remain, calls `ScheduleWatcherStop()`

---

## Terminal Lifecycle

The terminal system supports four distinct use cases, each with different
creation patterns, stream types, and cleanup behavior.

### Terminal types

| Type | Constant | Description |
|------|----------|-------------|
| **PTY** | `TypePTY = 1` | Pseudo-terminal with bidirectional I/O (keyboard input + screen output) |
| **Pipe** | `TypePipe = 0` | Unidirectional output stream (stdout/stderr only) |

### Creation matrix

| Use Case | Terminal Name | Type | Creation | Stream Source |
|----------|--------------|------|----------|---------------|
| Main shell | `"console"` | PTY | `Create()` + `StartPTY()` | `exec.Command("bash")` in stacks dir |
| Service exec | `"container-exec-{stack}-{svc}-0"` | PTY | `Recreate()` + `StartPTY()` | `docker compose exec {svc} {shell}` |
| Container exec | `"container-exec-by-name-{name}"` | PTY | `Create()` + `StartPTY()` | `docker exec -it {name} {shell}` |
| Container logs | `"container-log-{svc}"` | Pipe | `Recreate()` + `SetCancel()` | SDK `ContainerLogs` with follow |
| Combined logs | `"combined-{stack}"` | Pipe | `Create()` + `SetCancel()` | Per-container SDK streams merged |

### Combined logs architecture

Combined logs merge output from all containers in a stack into a single terminal
with colored, aligned prefixes:

```
┌─────────────────┐
│ ContainerList()  │  Get all containers for this stack
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────────────┐
│  Per-container goroutines                    │
│  readContainerLogs(container, lineCh)        │
│  Each reads SDK ContainerLogs (tail=100)     │
│  and sends formatted lines to shared channel │
└────────────────────┬────────────────────────┘
                     │
                     ▼
              lineCh (cap 256)
                     │
                     ▼
┌─────────────────────────────────────────────┐
│  flushLogLines() — batched flusher          │
│  50ms tick: drain lineCh → bytes.Buffer     │
│  Write batch to terminal (one WS message)   │
└─────────────────────────────────────────────┘
```

Each container gets a stable ANSI color from a palette of 6 colors (cyan, yellow,
green, magenta, blue, bright red). Service names are padded to the max length for
aligned output.

The combined log terminal subscribes to the EventBus for Docker events. On
container `"start"`, it spawns a new reader goroutine (guarded by `sync.Map` to
prevent duplicates) and injects a blue "started" banner. On `"die"`, it injects a
yellow "stopped" banner.

### EventBus reconnection for individual log streams

Individual container log streams (`runContainerLogLoop`) also subscribe to the
EventBus rather than opening their own Docker Events connection. When the
container dies, the log stream injects a stop banner and waits for the next
`"start"` event. On start, it re-resolves the container ID and reconnects with
`tail="0"` (new lines only).

### Buffer management

Every terminal has a rolling byte buffer:
- **Cap:** 64KB (65,536 bytes)
- **On overflow:** truncate to last 32KB (32,768 bytes)
- Pipe terminals normalize bare `\n` to `\r\n` for xterm rendering; PTY terminals
  skip this (the kernel's TTY discipline handles it)

The buffer is sent to newly joining clients via `JoinAndGetBuffer()`, which
atomically registers the writer and returns the buffer under a single lock to
prevent duplicate delivery.

### Cleanup

Three cleanup mechanisms ensure terminals don't leak:

**1. `RemoveAfter(name, 30s)`** — Scheduled after PTY process exit (exec
terminals) and after log stream end. A 30-second grace period allows the frontend
to re-join. If the terminal was recreated before the timer fires (pointer
comparison), the removal is a no-op.

**2. Explicit leave** — `leaveContainerLog` and `leaveCombinedTerminal` remove
the writer. If zero writers remain and the terminal has a cancel function
(indicating an active log stream), the terminal is removed and its context
cancelled.

**3. `RemoveWriterFromAll(connID)`** — Called on WebSocket disconnect. Two-phase:
first removes the writer from all terminals under RLock, collecting pipe terminals
that hit zero writers; then removes and closes those orphaned terminals under a
write lock.

### Key files

- `internal/terminal/manager.go` — Terminal types, buffer management, lifecycle
- `internal/handlers/terminal.go` — Handler layer: creation, log streaming, combined logs

---

## Docker CLI vs SDK

The backend uses two distinct interfaces to Docker, chosen by operation type:

### SDK (Go client) — All read operations

```go
type Client interface {
    ContainerList, ContainerListDetailed, ContainerInspect,
    ContainerStats, ContainerStartedAt, ContainerLogs, ContainerTop,
    ImageInspect, DistributionInspect, ImageList, ImageInspectDetail, ImagePrune,
    NetworkList, NetworkInspect,
    VolumeList, VolumeInspect,
    Events,
    Close,
}
```

The SDK provides structured data, streaming, and efficiency. A single
`SDKClient` wraps `github.com/docker/docker/client` and connects to whatever
`DOCKER_HOST` points to.

### CLI (`exec.Command`) — All write/compose operations

Compose lifecycle operations (up, down, stop, restart, pull) and interactive
sessions (exec) go through the Docker CLI as subprocesses:

- `docker compose up -d` — progress output streamed to PTY terminal
- `docker compose down` — with optional `--volumes` and `--remove-orphans`
- `docker compose stop/restart/pull` — progress output
- `docker compose exec {service} {shell}` — PTY multiplexed
- `docker exec -it {container} {shell}` — PTY multiplexed

### Rationale

The CLI is used for writes because:
- **Progress output**: `docker compose up` renders animated ANSI progress that
  users expect to see in the terminal
- **TTY multiplexing**: `docker exec` needs a real PTY for interactive shells
- **Env-file handling**: Compose reads `.env` files relative to the project
  directory, matching user expectations

The SDK is used for reads because:
- **Structured data**: JSON responses parse directly into Go structs
- **Streaming**: Events, logs, and stats stream efficiently over a single
  connection
- **No subprocess overhead**: Avoids fork/exec for frequent read operations

### Key files

- `internal/docker/docker.go` — Client interface definition
- `internal/docker/sdk.go` — SDK implementation
- `internal/handlers/stack.go` — CLI subprocess calls for compose operations

---

## Memory Optimization Techniques

The backend targets low memory consumption for self-hosted environments where
Dockge shares a machine with many other services. These techniques work together:

### GOMAXPROCS = 1

The default `--max-procs` is 1 (overridable via flag or `DOCKGE_MAX_PROCS` env).
This reduces per-P overhead: fewer mcache allocations, fewer `sync.Pool` shards,
and fewer idle gzip writers. For a single-user web app, one OS thread is
sufficient.

### Periodic `debug.FreeOSMemory()`

The Go runtime returns heap pages to the OS lazily. `FreeOSMemory()` forces
immediate return. It runs in three places:

1. **1-minute ticker** (main.go) — steady-state cleanup
2. **Idle transition** (broadcast.go) — when the watcher stops after 60s with no
   clients, aggressively reclaim before going dormant
3. **After `ContainerStats()`** (sdk.go) — stats collection allocates per-container
   response structs; reclaim the spike immediately

### `sync.Pool` for hot allocations

Two pools prevent repeated allocation of expensive objects:

| Pool | Object | Size | Location |
|------|--------|------|----------|
| `gzipPool` | `*gzip.Writer` | ~256KB internal state | main.go |
| `statsResponsePool` | `*container.StatsResponse` | ~2KB with nested maps | sdk.go |

Gzip writers are particularly expensive because each carries 256KB of internal
compression tables. Pooling them avoids reallocating on every HTTP response.
Stats response structs are zeroed and returned after each `ContainerStats()` call.

### Shared EventBus

A single `Docker.Events()` connection feeds all consumers through the EventBus
fan-out. Without this, each terminal log stream would open its own Events
connection (one HTTP streaming request per terminal), scaling linearly with
open terminals.

The EventBus uses non-blocking sends with buffer capacity 64 per subscriber.
If a subscriber falls behind, events are dropped rather than blocking the
producer.

### Terminal buffer capping

Terminal buffers are capped at 64KB with a keep-last-32KB overflow policy. This
bounds memory for long-running log streams that could otherwise accumulate
megabytes of output.

### Log line batch flushing

Combined log terminals coalesce individual log lines via a shared channel
(capacity 256) and a 50ms tick flusher. Instead of sending one WebSocket message
per log line per container, lines are batched into a `bytes.Buffer` (pre-grown to
4KB) and flushed as a single write. This reduces WebSocket frame overhead and
syscall frequency.

### No caching of dynamic data

The backend does not cache Docker state. Each broadcast queries Docker fresh,
with FNV-1a hashing preventing redundant sends. The Docker daemon is the single
source of truth. This eliminates an entire class of stale-cache bugs and avoids
the memory cost of maintaining shadow state.

Image update results are the one exception — they are cached in BoltDB because
registry checks are slow (seconds per image) and run on a 6-hour background
timer.

### Pre-marshaled JSON broadcasts

When broadcasting to N clients, the JSON payload is marshaled once (for FNV
hashing) and the same `[]byte` is written to every connection via
`BroadcastAuthenticatedBytes()`. This avoids N marshal operations that would each
allocate temporary buffers.

### Context timeouts on Docker API calls

All Docker API calls in broadcast functions use `context.WithTimeout(ctx, 15s)`.
This prevents a hung Docker daemon from leaking goroutines or holding connections
open indefinitely.

### Streaming I/O

Log output is piped from the Docker SDK directly to the WebSocket terminal
writer. The backend never buffers an entire log response before sending — data
flows through as it arrives.

---

## Mock System Architecture

The mock system enables development and E2E testing without a real Docker daemon.
It consists of three standalone binaries and a data-driven test fixture, with
**zero mock code in the production binary**.

### Separation principle

The dockge binary has no knowledge of mock mode. The mock system works entirely
through environment manipulation:

- `DOCKER_HOST=unix:///tmp/dockge-mock-$$/docker.sock` points the SDK client to
  the mock daemon
- `PATH=bin:$PATH` makes `exec.Command("docker", ...)` resolve to the
  mock docker CLI

The only production code exception is `POST /api/mock/reset`, a proxy handler in
main.go guarded by `--dev` flag with a Unix socket safety check.

### Components

```
test-data/stacks/
  ├── mock.yaml                    ← Global: standalone containers, external stacks
  ├── log-templates.yaml           ← Per-image log templates
  ├── test-alpine/
  │   ├── compose.yaml
  │   └── mock.yaml                ← status: running, per-service overrides
  ├── web-app/
  │   ├── compose.yaml
  │   └── mock.yaml
  └── ...

                ┌──────────────────────────┐
                │  BuildMockData()         │  Parses all compose.yaml + mock.yaml
                └────────────┬─────────────┘
                             │
                ┌────────────▼─────────────┐
                │   MockData (immutable)   │  Images, networks, volumes, services,
                │                          │  update flags, log templates
                └────────────┬─────────────┘
                             │
          ┌──────────────────┼──────────────────┐
          │                  │                  │
   ┌──────▼──────┐    ┌─────▼──────┐    ┌──────▼──────────┐
   │  MockState  │    │ MockWorld  │    │  seed-testdb    │
   │  (mutable)  │    │ (live view │    │  SeedFromMock() │
   │  stack →    │    │  built from│    │  → BoltDB       │
   │  status map │    │  Data +    │    └─────────────────┘
   └──────┬──────┘    │  State)    │
          │           └─────┬──────┘
          └────────┬────────┘
                   │
          ┌────────▼────────┐
          │  FakeDaemon     │  Unix socket HTTP server
          │  Docker Engine  │  implementing the Docker API
          │  API            │
          └────────┬────────┘
                   │ DOCKER_HOST
          ┌────────▼────────┐         ┌──────────────────┐
          │  dockge binary  │◄───────►│  mock-docker CLI │
          │  (SDKClient)    │  exec   │  (on PATH)       │
          └─────────────────┘         │  /_mock/state/*  │
                                      └──────────────────┘
```

### Mock daemon (`cmd/mock-daemon`)

Standalone process that serves a fake Docker Engine API:

1. Calls `BuildMockData(stacksDir)` to parse all test stack definitions
2. Calls `DefaultDevStateFromData(mockData)` to create the initial `MockState`
3. Builds a `MockWorld` — a materialized view of live Docker state from Data + State
4. Calls `StartFakeDaemonOnSocket()` to serve HTTP on the Unix socket

The fake daemon implements the Docker Engine API routes that the SDK client uses:
`/containers/json`, `/containers/{id}/json`, `/containers/{id}/stats`,
`/containers/{id}/logs`, `/images/json`, `/networks`, `/volumes`, `/events`,
`/distribution/{name}/json`, and more.

For image update simulation, `handleDistributionInspect()` returns a different
registry digest when `HasUpdateAvailable(imageRef)` is true, making the backend's
digest comparison detect a "new version available."

### Mock docker CLI (`cmd/mock-docker`)

Masquerades as the `docker` CLI. When the backend calls
`exec.Command("docker", "compose", "up", ...)`, this binary:

1. Renders animated ANSI progress output faithfully reproducing Docker Compose v2
   terminal UI (spinners, checkmarks, elapsed times)
2. Sends `POST /_mock/state/{stack}` to the mock daemon over the Unix socket to
   update the in-memory state
3. The daemon updates MockState, rebuilds MockWorld, and publishes Docker events
   to all subscribers

The mock-docker CLI is stateless — it communicates all state changes back to the
daemon via HTTP.

### BoltDB seeder (`cmd/seed-testdb`)

Runs once after the mock daemon starts to pre-populate BoltDB:

1. Creates admin user (`admin` / `testpass123`)
2. Ensures a JWT secret exists
3. Seeds image update flags from `MockData.UpdateFlags()` into BoltDB
4. Stamps `imageUpdateLastCheck` to prevent the background checker from running
   immediately

### Mock reset flow

`POST /api/mock/reset` (dev mode only) enables E2E tests to restore pristine
state between test runs:

```
POST /api/mock/reset
  → backend proxies to daemon's /_mock/reset
  → daemon restores files from pristine source copy
  → daemon rebuilds MockData + MockState + MockWorld
  → daemon returns updateFlags in response body
  → backend re-seeds BoltDB image updates from flags
  → backend triggers all 6 broadcast channels
  → frontend receives fresh state
```

### Task orchestration

`task dev` orchestrates the full startup sequence:

1. **Build** — compiles 4 binaries into `bin/`: `dockge`, `mock-daemon`, `docker`, `seed-testdb`
2. **Start mock daemon** — launches on a Unix socket, polls until ready (up to 5s)
3. **Seed database** — runs `seed-testdb` to populate BoltDB
4. **Set environment** — exports `DOCKER_HOST` and prepends mock binaries to `PATH`
5. **Start servers** — launches Vite (HMR on :5000) and dockge (backend on :5001) in parallel

### Key files

- `cmd/mock-daemon/main.go` — Mock daemon entry point
- `cmd/mock-docker/main.go` — Mock docker CLI
- `cmd/seed-testdb/main.go` — BoltDB seeder
- `internal/docker/mock/mockdata.go` — Test data parsing from `test-data/stacks/`
- `internal/docker/mock/mockstate.go` — In-memory mutable state
- `internal/docker/mock/mockworld.go` — Materialized live Docker environment
- `internal/docker/mock/fakedaemon.go` — Fake Docker Engine API server
- `internal/models/image_update.go` — BoltDB image update cache + `SeedFromMock()`
