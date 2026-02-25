# Contributing to Dockge (cfilipov fork)

## Architecture

- **Frontend**: Vue.js 3 + Bootstrap 5, built with Vite, communicates via WebSocket
- **Backend**: Go (root module), using BoltDB for persistence, `coder/websocket` for real-time communication
- **Stacks**: Plain `compose.yaml` + `.env` files on disk (`/opt/stacks/`), not in the database

### Project layout

```
go.mod                           # Go module (github.com/cfilipov/dockge)
main.go                          # Entry point, HTTP server, WebSocket upgrade
embed.go                         # //go:embed all:dist
Taskfile.yml                     # Build orchestration (run `task --list`)
internal/
  config/                        # CLI flags and env var parsing
  db/                            # BoltDB wrapper
  docker/                        # Docker/Compose command interfaces + mock
  handlers/                      # WebSocket event handlers
  models/                        # User, Setting, Agent, ImageUpdate stores
  compose/                       # YAML parser, compose file resolution, ComposeCache
  stack/                         # Stack model (status, file I/O, JSON serialization)
  terminal/                      # PTY/pipe terminal manager
  updatechecker/                 # Background image update checker
  ws/                            # WebSocket protocol types
  testutil/                      # Integration test harness
web/                             # Vue 3 frontend (self-contained Node project)
  package.json
  vite.config.ts
  src/
    pages/                       # Vue page components
    components/                  # Reusable Vue components
    composables/                 # Composition API composables
    common/                      # Shared TypeScript types and utilities
    lang/                        # i18n translations
    styles/                      # SCSS (vars, main, localization)
e2e/                             # Playwright E2E tests (own package.json)
  tests/                         # Test specs
  __screenshots__/               # Golden screenshots (committed)
test-data/
  stacks/                        # Mock stacks for dev/test
  dockge-bolt.db                 # BoltDB file (gitignored, created at runtime)
docker/Dockerfile                # Production multi-stage build
.github/
  workflows/                     # CI pipelines
  scripts/                       # CI utility scripts
```

## Prerequisites

- [Go](https://go.dev/dl/) 1.25+
- [Node.js](https://nodejs.org/) 22+ and [pnpm](https://pnpm.io/)
- [Task](https://taskfile.dev/) (installed by `bootstrap.sh`, or see [manual install](#installing-task-manually))

## Getting started

### Linux / macOS

```bash
./bootstrap.sh                   # Installs Task and runs setup (deps + Playwright)
task dev                         # Go backend (5001) + Vite HMR (5000)
```

### Windows

```powershell
go install github.com/go-task/task/v3/cmd/task@v3.45.3
task setup                       # Install deps + Playwright
task dev                         # Go backend (5001) + Vite HMR (5000)
```

Ctrl+C stops both. Use port 5000 for development — Vite proxies `/ws` to the backend automatically.

No real Docker daemon is needed. The `--mock` flag provides in-memory Docker state with four seeded stacks. Dev data (BoltDB) is stored in `test-data/`.

## Task targets

Run `task --list` to see all targets. The important ones:

| Target | Description |
|--------|-------------|
| `task setup` | Install all dependencies (first-time dev setup) |
| `task build` | Build everything (frontend + Go binary) |
| `task dev` | Run Go backend + Vite HMR concurrently |
| `task dev-go` | Run Go backend only (port 5001) |
| `task dev-web` | Run Vite dev server only (port 5000) |
| `task kill` | Kill any running backend or Vite processes |
| `task test` | Run all tests (Go + E2E) |
| `task check-ts` | TypeScript type check (`tsc --noEmit`) |
| `task lint` | Lint frontend (ESLint) and Go (`go vet`) |
| `task fmt` | Format frontend and Go (`gofmt`) |
| `task clean` | Remove build artifacts |
| `task test-e2e-report` | Show Playwright HTML report with screenshot diffs |
| `task update-screenshots` | Update E2E golden screenshots |
| `task docker` | Build production Docker image |

## Building

```bash
task build
```

This builds the frontend (`dist/`) and the Go binary (`dockge`). The binary is self-contained — in production it embeds the frontend via `embed.FS`.

## Development

### CLI flags

The task targets handle these, but for reference:

| Flag | Default | Env var | Description |
|------|---------|---------|-------------|
| `--port` | `5001` | `DOCKGE_PORT` | HTTP server port |
| `--stacks-dir` | `/opt/stacks` | `DOCKGE_STACKS_DIR` | Path to stacks directory |
| `--data-dir` | `./data` | `DOCKGE_DATA_DIR` | Path to BoltDB data. Dev uses `test-data/`. |
| `--dev` | `false` | — | Serve frontend from `dist/` on disk. With `--mock`, seeds admin user (`admin`/`testpass123`). Enables pprof. |
| `--mock` | `false` | `DOCKGE_MOCK=1` | In-memory mock Docker — no daemon needed. State is lost on restart. |
| `--log-level` | `info` | `DOCKGE_LOG_LEVEL` | `debug`, `info`, `warn`, or `error` |
| `--no-auth` | `false` | `DOCKGE_NO_AUTH=1` | Disable authentication |

### Mock test stacks

Located in `test-data/stacks/`:

| Stack | Services | Notes |
|-------|----------|-------|
| `test-alpine` | alpine | Single service |
| `web-app` | nginx, redis | Two services, port 8080 |
| `monitoring` | grafana | Single service |
| `blog` | wordpress, mysql | Triggers recreateNecessary |

## Running tests

```bash
task test                        # All tests (Go + E2E)
task test-go                     # Go tests with race detector
task test-e2e                    # Playwright E2E tests (builds frontend + backend first)
```

### Visual regression screenshots

Golden screenshots are committed to `e2e/__screenshots__/`. When a screenshot test fails, Playwright writes expected/actual/diff images to `e2e/test-results/`. View the HTML report:

```bash
task test-e2e-report
```

#### Updating golden screenshots

If a UI change intentionally alters how pages look, update the golden screenshots:

```bash
task update-screenshots
```

This rebuilds the frontend and backend, then re-runs all E2E tests with `--update-snapshots` to regenerate the golden files. Review the diff in `e2e/__screenshots__/` before committing — every changed screenshot should correspond to an intentional UI change. If a screenshot changed unexpectedly, investigate before committing.

**Design notes:**
- Tests run sequentially (`workers: 1`) — the mock backend has shared mutable state
- All UI state is deterministic via the mock backend
- Auth uses Playwright's `storageState` pattern

### Adding test stacks

Place a `compose.yaml` in `test-data/stacks/<name>/`.

## Coding style

- **Go**: `gofmt` (run `task fmt`)
- **TypeScript/Vue**: 4 spaces, camelCase (run `task lint`)
- **CSS/SCSS**: kebab-case
- **BoltDB keys**: snake_case

## Commit messages

Prefix with the type of change:

- `[feature]` — new features
- `[fix]` — bug fixes
- `[cleanup]` — removing or tidying code
- `[hamphh]` — features ported from the hamphh fork

Do **not** include `Co-Authored-By` footers or other trailers.

## Performance rules

Performance is the top priority:

- Never block request handlers on Docker commands or registry lookups
- Image update checks run on a background timer (default: 6 hours), never on page load
- Results are cached in BoltDB; the UI reads cached state only
- No polling loops or `setInterval` in the frontend — the backend pushes via WebSocket

## Translations

Add translatable strings to `web/src/lang/en.json`.

## Dependencies

- **Frontend**: `web/package.json`
- **E2E tests**: `e2e/package.json`
- **Go backend**: `go.mod` (project root)

## Installing Task manually

If you can't run `bootstrap.sh`, install [Task](https://taskfile.dev/) and run setup yourself:

**With Go** (all platforms):
```bash
go install github.com/go-task/task/v3/cmd/task@v3.45.3
task setup
```

**Linux/macOS** (install script):
```bash
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin
task setup
```

**Windows** (Chocolatey, Scoop, or Winget):
```powershell
choco install go-task        # Chocolatey
scoop install task           # Scoop
winget install Task.Task     # Winget
```

Then run `task setup` to install dependencies.

See the [official installation docs](https://taskfile.dev/docs/installation) for more options.
