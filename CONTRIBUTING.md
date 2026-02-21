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

- [Go](https://go.dev/dl/) 1.25+
- [Node.js](https://nodejs.org/) 22+ and [pnpm](https://pnpm.io/) (for the frontend)
- [git](https://git-scm.com/)
- Chromium for Playwright (for E2E tests) — installed via `npx playwright install chromium` after `pnpm install`

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

### Running in dev mode

The standard way to develop is with both `--dev` and `--mock` enabled. No real Docker daemon is needed.

**Start the Go backend in dev mode:**

```bash
cd backend-go
go build -o dockge-backend . && ./dockge-backend --dev --mock --port 5001 --stacks-dir test-data/stacks
```

### CLI flags

| Flag | Default | Env var | Description |
|------|---------|---------|-------------|
| `--port` | `5001` | `DOCKGE_PORT` | HTTP server port |
| `--stacks-dir` | `/opt/stacks` | `DOCKGE_STACKS_DIR` | Path to stacks directory |
| `--data-dir` | `./data` | `DOCKGE_DATA_DIR` | Path to data directory (BoltDB) |
| `--dev` | `false` | — | Serves frontend from `../frontend-dist/` on disk instead of the embedded `embed.FS`, so you can rebuild the frontend and refresh without restarting the backend. When combined with `--mock` on an empty database, auto-seeds an admin user (`admin`/`testpass123`). Also enables `/debug/pprof/` endpoints. |
| `--mock` | `false` | `DOCKGE_MOCK=1` | Uses in-memory `MockClient` and `MockCompose` instead of the real Docker SDK — no Docker daemon needed. `DefaultDevState()` seeds four stacks (`web-app`, `monitoring`, `test-alpine`, `blog`) with running/exited statuses. State is lost on restart. |
| `--log-level` | `info` | `DOCKGE_LOG_LEVEL` | Log level: `debug`, `info`, `warn`, or `error`. |
| `--no-auth` | `false` | `DOCKGE_NO_AUTH=1` | Disable authentication — all WebSocket endpoints are open without login. Useful for development and testing. |

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

**Test stacks** in `backend-go/test-data/stacks/`:

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

### E2E tests (Playwright)

E2E tests live in `e2e/` and use Playwright with Chromium. They run against the Go backend in `--dev --mock` mode (started automatically) and cover both functional assertions and visual regression via screenshot comparison.

#### First-time setup

After `pnpm install`, you need to download the Chromium browser binary once:

```bash
PLAYWRIGHT_BROWSERS_PATH=$HOME/.cache/ms-playwright npx playwright install chromium
```

This downloads ~280 MB to `~/.cache/ms-playwright/`. You only need to redo this when upgrading `@playwright/test`.

#### Running e2e tests

The test runner automatically builds the Go backend and starts it in `--dev --mock` mode. Make sure the frontend is built first:

```bash
# Build the frontend (backend is built automatically by the test runner)
pnpm run build:frontend

# Run all E2E tests
pnpm run test:e2e

# Run with visible browser
pnpm run test:e2e:headed

# Open Playwright UI mode (interactive)
pnpm run test:e2e:ui
```

#### Visual regression screenshots

Golden screenshots are committed to `e2e/__screenshots__/`. Playwright compares each test's screenshot against its golden file and fails if they differ beyond the configured threshold.

**When your changes intentionally alter the UI:**

```bash
# Re-generate all golden screenshots
pnpm run test:e2e:update-screenshots

# Verify the new screenshots pass
pnpm run test:e2e
```

Always review the updated screenshots before committing — `git diff` won't show image changes, so open the PNGs in `e2e/__screenshots__/` and verify they look correct.

**When a screenshot test fails unexpectedly:**

Playwright writes three images to `e2e/test-results/`:
- `*-expected.png` — the golden file
- `*-actual.png` — what the test captured
- `*-diff.png` — red highlights showing the differences

Open the HTML report for a side-by-side view:

```bash
pnpm run test:e2e:report
```

**Key design notes:**
- Tests run sequentially (`workers: 1`) because the mock backend has shared mutable state
- All UI state is deterministic (mock backend provides fixed statuses, icons, etc.)
- Auth is handled via Playwright's `storageState` pattern — `auth.setup.ts` logs in once and all other tests reuse the saved session

### Adding test stacks

Place a `compose.yaml` in `backend-go/test-data/stacks/<name>/`. Tests automatically copy these into isolated temp directories.

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
