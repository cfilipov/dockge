# Contributing to Dockge (cfilipov fork)

## Architecture

- **Frontend**: Vue.js 3 + Bootstrap 5, built with Vite, communicates via WebSocket
- **Backend**: Go (`backend-go/`), using BoltDB for persistence, `coder/websocket` for real-time communication
- **Stacks**: Plain `compose.yaml` + `.env` files on disk (`/opt/stacks/`), not in the database

### Key directories

```
backend-go/
  main.go                      # Entry point, HTTP server, WebSocket upgrade
  internal/
    config/                    # CLI flags and env var parsing
    db/                        # BoltDB wrapper
    docker/                    # Docker/Compose command interfaces + mock
    handlers/                  # WebSocket event handlers (auth, stack, service, settings, agent)
    models/                    # User, Setting, Agent, ImageUpdate stores (BoltDB)
    compose/                   # YAML parser, compose file resolution, ComposeCache
    stack/                     # Stack model (status, file I/O, JSON serialization)
    terminal/                  # PTY/pipe terminal manager
    updatechecker/             # Background image update checker (registry digest comparison)
    ws/                        # WebSocket protocol types
    testutil/                  # Integration test harness (TestEnv, MockDocker, MockCompose)
frontend/
  src/
    pages/                     # Vue page components (Compose.vue, DashboardHome.vue)
    components/                # Reusable Vue components
    lang/                      # i18n translations
common/                        # Shared types between frontend and backend (TypeScript)
```

## Prerequisites

- [Go](https://go.dev/dl/) 1.24+
- [Node.js](https://nodejs.org/) 22+ and [pnpm](https://pnpm.io/) (for the frontend)
- [git](https://git-scm.com/)

## Building

### Backend

```bash
cd backend-go
go build -o dockge-backend .
```

The binary is self-contained. In production it embeds the frontend via `embed.FS`.

### Frontend

```bash
pnpm install
pnpm run build:frontend
```

This outputs to `frontend-dist/`, which the Go backend serves.

### Full production build

```bash
pnpm install
pnpm run build:frontend
cd backend-go && go build -o dockge-backend .
```

## Development

### Dev mode with mock Docker

The project includes a mock Docker CLI that simulates container lifecycle without a real Docker daemon. This is the standard way to develop.

**Start the Go backend in dev mode:**

```bash
cd backend-go
go build -o dockge-backend . && ./dockge-backend --dev --mock --port 5001 --stacks-dir /opt/stacks
```

In `--dev` mode the backend serves the frontend from `../frontend-dist/` on the filesystem (not embedded), so you can rebuild the frontend and refresh without restarting the backend.

When `--dev --mock` are both set and the database is empty, an admin user (`admin`/`testpass123`) is created automatically.

### CLI flags

| Flag | Default | Env var | Description |
|------|---------|---------|-------------|
| `--port` | `5001` | `DOCKGE_PORT` | HTTP server port |
| `--stacks-dir` | `/opt/stacks` | `DOCKGE_STACKS_DIR` | Path to stacks directory |
| `--data-dir` | `./data` | `DOCKGE_DATA_DIR` | Path to data directory (BoltDB) |
| `--dev` | `false` | — | Serve frontend from filesystem instead of embedded |
| `--mock` | `false` | `DOCKGE_MOCK=1` | Use mock Docker CLI instead of Docker SDK |

Environment variables override flags if set.

### Vite dev server (optional, for frontend HMR)

```bash
pnpm run dev:frontend
```

This runs Vite on port 5000 with hot module replacement. The Vite config proxies `/socket.io/` requests to the Go backend on port 5001.

After making frontend changes, rebuild the bundle for the backend to serve on port 5001:

```bash
pnpm run build:frontend
```

Port 5000 (Vite HMR) reflects changes immediately. Port 5001 serves the pre-built bundle.

### Mock Docker mode

When `--mock` is set, the Go backend uses in-memory `MockClient` and `MockCompose` implementations instead of the real Docker SDK. Both share a `MockState` map of stack statuses. `DefaultDevState()` seeds four stacks on startup; state is lost on restart.

**Test stacks** in `/opt/stacks/`:

| Stack | Services | Notes |
|-------|----------|-------|
| `test-alpine` | alpine | Single service |
| `web-app` | nginx, redis | Two services, port 8080 |
| `monitoring` | grafana | Single service |
| `blog` | wordpress, mysql | Triggers recreateNecessary |

## Running tests

Tests use an in-process HTTP server with real WebSocket connections and mock Docker — no Docker daemon or external services needed.

### All tests with race detector (recommended)

```bash
cd backend-go
go test -race ./...
```

### Verbose output

```bash
cd backend-go
go test -race -v ./...
```

### Coverage report

```bash
cd backend-go
go test -coverpkg=./... -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

### Fuzz tests

```bash
cd backend-go
go test -fuzz=FuzzParseYAML -fuzztime=10s ./internal/compose/
```

### Benchmarks

```bash
cd backend-go
go test -bench=. -benchmem -run='^$' .
```

### Frontend checks

```bash
pnpm run lint
pnpm run check-ts
pnpm run build:frontend
```

### Adding test stacks

Place a `compose.yaml` in `backend-go/testdata/stacks/<name>/`. Tests automatically copy these into isolated temp directories.

## Coding style

- **Go**: standard `gofmt`
- **TypeScript/Vue**: 4 spaces, camelCase
- **CSS/SCSS**: kebab-case
- **BoltDB keys**: snake_case
- Follow `.editorconfig` and ESLint

## Commit messages

Use descriptive commit messages with a prefix:

- `[feature]` — new features
- `[fix]` — bug fixes
- `[cleanup]` — removing or tidying code
- `[hamphh]` — features ported from the hamphh fork

Do **not** include `Co-Authored-By` footers or other trailers.

## Performance rules

Performance is the top priority. All changes must follow these rules:

- The stack list must load instantly — never block on update checks or registry lookups
- All Docker and registry API calls must be async and non-blocking
- Image update checks run on a background timer (default: every 6 hours), never on page load
- Results are cached in BoltDB; the UI reads cached state only
- Never add polling loops or `setInterval` calls to the frontend

## Translations

Please add all translatable strings to `frontend/src/lang/en.json`. Don't include other languages in your initial PR to avoid merge conflicts.

## Dependencies

Frontend and backend share the same `package.json` for the Node.js/Vue toolchain:

- Frontend dependencies = `devDependencies` (vue, chart.js, etc.)
- Development tooling = `devDependencies` (eslint, sass, vite, etc.)

Go backend dependencies are managed separately via `backend-go/go.mod`.
