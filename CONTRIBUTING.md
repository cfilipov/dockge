## Can I create a pull request for Dockge?

Yes or no, it depends on what you will try to do. Since I don't want to waste your time, be sure to **create open a discussion, so we can have a discussion first**. Especially for a large pull request or you don't know if it will be merged or not.

Here are some references:

### ✅ Usually accepted:
- Bug fix
- Security fix
- Adding new language files (see [these instructions](https://github.com/louislam/dockge/blob/master/frontend/src/lang/README.md))
- Adding new language keys: `$t("...")`

### ⚠️ Discussion required:
- Large pull requests
- New features

### ❌ Won't be merged:
- A dedicated PR for translating existing languages (see [these instructions](https://github.com/louislam/dockge/blob/master/frontend/src/lang/README.md))
- Do not pass the auto-test
- Any breaking changes
- Duplicated pull requests
- Buggy
- UI/UX is not close to Dockge
- Modifications or deletions of existing logic without a valid reason.
- Adding functions that is completely out of scope
- Converting existing code into other programming languages
- Unnecessarily large code changes that are hard to review and cause conflicts with other PRs.

The above cases may not cover all possible situations.

I (@louislam) have the final say. If your pull request does not meet my expectations, I will reject it, no matter how much time you spend on it. Therefore, it is essential to have a discussion beforehand.

I will assign your pull request to a [milestone](https://github.com/louislam/dockge/milestones), if I plan to review and merge it.

Also, please don't rush or ask for an ETA, because I have to understand the pull request, make sure it is no breaking changes and stick to my vision of this project, especially for large pull requests.

## Project Styles

I personally do not like something that requires so many configurations before you can finally start the app.

- Settings should be configurable in the frontend. Environment variables are discouraged, unless it is related to startup such as `DOCKGE_STACKS_DIR`
- Easy to use
- The web UI styling should be consistent and nice
- No native build dependency

## Coding Styles

- 4 spaces indentation
- Follow `.editorconfig`
- Follow ESLint
- Methods and functions should be documented with JSDoc

## Name Conventions

- Javascript/Typescript: camelCaseType
- SQLite: snake_case (Underscore)
- CSS/SCSS: kebab-case (Dash)

## Tools

- [`Node.js`](https://nodejs.org/) >= 22.14.0
- [`git`](https://git-scm.com/)
- IDE that supports [`ESLint`](https://eslint.org/) and EditorConfig (I am using [`IntelliJ IDEA`](https://www.jetbrains.com/idea/))
- A SQLite GUI tool (f.ex. [`SQLite Expert Personal`](https://www.sqliteexpert.com/download.html) or [`DBeaver Community`](https://dbeaver.io/download/))

## Install Dependencies for Development

```bash
pnpm install
```

## Dev Server

The recommended way to develop is with the mock Docker CLI, which simulates Docker without requiring a real daemon. This is especially useful in environments without Docker-in-Docker support.

### Quick Start (with mock Docker)

```bash
# 1. Install dependencies
pnpm install

# 2. Set up test stacks and mock state
#    (requires /opt/stacks to exist and be writable)
sudo mkdir -p /opt/stacks && sudo chown $USER /opt/stacks
./extra/mock-docker/setup-mock-env.sh

# 3. Start the dev server with mock Docker on the PATH
PATH="$PWD/extra/mock-docker:$PATH" DOCKGE_STACKS_DIR=/opt/stacks pnpm run dev
```

This starts:
- **Vite frontend** on `http://localhost:5000` (hot reload)
- **Express backend** on `http://localhost:5001` (serves API + Socket.IO)

### Quick Start (with real Docker)

If you have a real Docker daemon available:

```bash
pnpm install
pnpm run dev
```

### Mock Docker CLI

The mock Docker CLI at `extra/mock-docker/docker` intercepts all `docker` and `docker compose` commands. It simulates container lifecycle by writing state files to `/tmp/mock-docker/state/`.

**Supported commands:**
- `docker compose ls/ps/up/stop/down/restart/pull/config/logs`
- `docker inspect`, `docker stats`, `docker network ls`, `docker version`
- `docker image inspect`, `docker image prune`

**How it works:**
- Stack state (running/exited) is stored in `/tmp/mock-docker/state/<stackname>/status`
- Service names are parsed from the actual `compose.yaml` files in the stacks directory
- `compose up` sets status to `running`, `stop` sets to `exited`, `down` removes state
- `compose logs -f` outputs mock lines with periodic heartbeats
- `compose ps --format json` returns mock container JSON for running stacks
- Some stacks have hardcoded quirks for testing (e.g., `web-app/redis` is always exited to test mixed-status stacks)

**Test stacks** are in `extra/test-stacks/` and get copied to `/opt/stacks/` by the setup script:

| Stack | Services | Default State | Has Update |
|-------|----------|---------------|------------|
| `web-app` | nginx, redis | running | Yes (nginx) |
| `blog` | wordpress, mysql | running | Yes (wordpress) |
| `test-alpine` | alpine | exited | No |
| `monitoring` | grafana | exited | No |
| `database` | postgres | exited | Yes |
| `cache` | redis | down | No |

**Adding a new test stack:** Create a directory in `extra/test-stacks/<name>/` with a `compose.yaml`, then re-run `setup-mock-env.sh`.

**Adding new mock commands:** Edit `extra/mock-docker/docker`. Add a `cmd_*` function and wire it into the routing case statements at the bottom.

### Build the frontend

After making frontend changes, rebuild the bundle for the backend to serve:

```bash
pnpm run build:frontend
```

Port 5000 (Vite HMR) reflects changes immediately. Port 5001 serves the pre-built bundle.

## Backend Dev Server (Node.js)

It binds to `0.0.0.0:5001` by default.

It is mainly a socket.io app + express.js.

## Backend Dev Server (Rust) — Experimental

An alternative backend written in Rust is available in `backend-rust/`. It implements the same Socket.IO protocol as the Node.js backend, so the frontend works with either one.

### Prerequisites

- [Rust toolchain](https://rustup.rs/) (stable, 1.75+)
- No other dependencies — everything is statically linked

### Building

```bash
# Debug build (fast compile, large binary ~182MB, includes debug symbols)
cd backend-rust
cargo build

# Release build (optimized, ~11MB binary)
cargo build --release
```

### Running

```bash
# With the mock Docker CLI:
PATH="$PWD/extra/mock-docker:$PATH" \
  ./backend-rust/target/release/dockge-backend \
  --port 5002 \
  --stacks-dir /opt/stacks

# With real Docker (just omit the PATH override):
./backend-rust/target/release/dockge-backend \
  --port 5002 \
  --stacks-dir /opt/stacks
```

The Rust backend uses its own data directory (`backend-rust/data/` by default) with its own SQLite database and JWT secret. This means you'll need to create a new admin account the first time you connect.

### Using with the Vite frontend

To point the Vite dev server at the Rust backend instead of Node.js, change the proxy target in `frontend/vite.config.ts`:

```typescript
proxy: {
    "/socket.io/": {
        target: "http://localhost:5002",  // Rust backend
        // target: "http://localhost:5001",  // Node.js backend
        ws: true,
    },
},
```

Then access the frontend at `http://localhost:5000` as usual.

### CLI options

```
Options:
  --port <PORT>            Port to listen on [default: 5001]
  --hostname <HOSTNAME>    Hostname to bind to [default: 0.0.0.0]
  --data-dir <DATA_DIR>    Data directory [default: ./data/]
  --stacks-dir <PATH>      Stacks directory [default: /opt/stacks]
  --enable-console         Enable the web terminal console
```

### Architecture

The Rust backend uses:
- **axum** for HTTP/WebSocket serving
- **socketioxide** for Socket.IO protocol
- **sqlx** with SQLite for database
- **portable-pty** for real PTY terminals (compose actions, interactive exec, logs)
- **tokio** async runtime

### Current limitations

- Agent-to-agent communication is stubbed (local stacks only)
- No SSL/TLS support (use a reverse proxy)
- No Docker image management feature

## Backend Dev Server (Go) — Experimental

An alternative backend written in Go is available in `backend-go/`. It implements the same WebSocket protocol as the Node.js backend, so the frontend works with either one.

### Prerequisites

- [Go](https://go.dev/dl/) 1.25+

### Building

```bash
cd backend-go
go build -o dockge-backend .
```

### Running

```bash
# With mock Docker (no real Docker daemon needed):
./dockge-backend --dev --mock --port 5002 --stacks-dir /opt/stacks

# With real Docker:
./dockge-backend --port 5002 --stacks-dir /opt/stacks
```

When `--dev --mock` are both set and the database is empty, an admin user (`admin`/`testpass123`) is created automatically.

### Running Tests

Tests use an in-process HTTP server with real WebSocket connections and mock Docker — no Docker daemon or external services needed.

```bash
cd backend-go

# Quick correctness tests (skips binary size and memory budget checks):
go test -short -v ./...

# Full test suite including binary size and memory budget:
go test -v -count=1 ./...

# Run benchmarks only (for regression tracking):
go test -bench=. -benchmem -count=5 -run='^$' .

# Compare benchmarks against committed baseline:
# (requires golang.org/x/perf/cmd/benchstat)
go test -bench=. -benchmem -count=5 -run='^$' . 2>/dev/null \
  | grep -E "(^goos:|^goarch:|^pkg:|^cpu:|ns/op)" > bench-new.txt
benchstat testdata/benchmarks/baseline.txt bench-new.txt

# Update baseline after intentional changes:
cp bench-new.txt testdata/benchmarks/baseline.txt
```

**Test categories:**

| Test | What it checks |
|------|---------------|
| `TestNeedSetup` | Fresh DB returns needSetup=true |
| `TestSetupAndLogin` | Create user, then login succeeds |
| `TestLoginBadPassword` | Wrong password returns ok=false |
| `TestRequestStackList` | Returns stack list without error |
| `TestGetStack` | Returns YAML content from disk |
| `TestSaveStack` | Saves YAML, verifies file on disk |
| `TestDockerStats` | Returns stats for a running stack |
| `TestServiceStatusList` | Returns container statuses |
| `TestGetDockerNetworkList` | Returns network names |
| `TestUnauthenticatedAccess` | Protected endpoints reject unauthenticated requests |
| `TestBinarySize` | Production binary stays under 20 MB |
| `TestMemoryBudget` | Heap stays under 50 MB after a workload |

**Benchmarks:** `BenchmarkLogin`, `BenchmarkRequestStackList`, `BenchmarkGetStack`, `BenchmarkServiceStatusList`, `BenchmarkDockerStats`

### Adding Test Stacks

Place a `compose.yaml` in `backend-go/testdata/stacks/<name>/`. Tests automatically copy these into isolated temp directories.

## Frontend Dev Server

It binds to `0.0.0.0:5000` by default. The frontend dev server is used for development only.

For production, it is not used. It will be compiled to `frontend-dist` directory instead.

You can use Vue.js devtools Chrome extension for debugging.

## Database Migration

TODO

## Dependencies

Both frontend and backend share the same package.json. However, the frontend dependencies are eventually not used in the production environment, because it is usually also baked into dist files. So:

- Frontend dependencies = "devDependencies"
    - Examples: vue, chart.js
- Backend dependencies = "dependencies"
    - Examples: socket.io, sqlite3
- Development dependencies = "devDependencies"
    - Examples: eslint, sass

### Update Dependencies

Should only be done by the maintainer.

```bash
npm update
````

It should update the patch release version only.

Patch release = the third digit ([Semantic Versioning](https://semver.org/))

If for security / bug / other reasons, a library must be updated, breaking changes need to be checked by the person proposing the change.

## Translations

Please add **all** the strings which are translatable to `src/lang/en.json` (If translation keys are omitted, they can not be translated).

**Don't include any other languages in your initial Pull-Request** (even if this is your mother tongue), to avoid merge-conflicts between weblate and `master`.  
The translations can then (after merging a PR into `master`) be translated by awesome people donating their language skills.

If you want to help by translating Uptime Kuma into your language, please visit the [instructions on how to translate using weblate](https://github.com/louislam/uptime-kuma/blob/master/src/lang/README.md).

## Spelling & Grammar

Feel free to correct the grammar in the documentation or code.
My mother language is not English and my grammar is not that great.
