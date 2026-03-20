# Dev Workflow

## Fixed ports — never deviate

| Service | Port | Command |
|---------|------|---------|
| Vue Vite dev (HMR) | 5000 | `task dev-web` / `task dev-rust-web` |
| Go backend | 5001 | `task dev-go` |
| Rust backend | 5003 | `task dev-rust-backend` |
| E2E test backend | 5052 | `task test-e2e` |
| Svelte Vite dev (HMR) | 6100 | `task dev-svelte-web` |
| Go backend (Svelte dev) | 6001 | `task dev-svelte-go` |
| Storybook | 6200 | `cd web-svelte && npx storybook dev -p 6200` |

If a port is in use, investigate and reuse the existing process. Never start on a different port.

### Keep dev servers running during work (MANDATORY)

**CRITICAL: Start `task dev` (or `task dev-go` + `task dev-web` separately) at the BEGINNING of the session and NEVER stop them unless the user explicitly asks.** The user tests changes in real time through the Vite dev server — killing it blocks their workflow. This is non-negotiable:

- Start both servers (`task dev-go` on :5001, `task dev-web` on :5000) before making any code changes
- Vite HMR hot-reloads frontend changes automatically — do NOT run `task build-web` to test frontend changes, the user sees them instantly via Vite on port 5000
- If the Go backend needs rebuilding (handler changes, etc.), restart ONLY `task dev-go` — NEVER kill the Vite server
- When you finish your work, leave BOTH servers running — do NOT run `task kill`
- Only run `task kill` if the user explicitly asks to stop the servers

## Browser automation

Use `agent-browser` for all browser interactions. Do NOT use Playwright MCP tools (`mcp__playwright__*`) unless `agent-browser` is insufficient and you need specific capabilities like `browser_evaluate`.

```bash
agent-browser open http://localhost:5000/
agent-browser snapshot -i
agent-browser fill @e1 "admin"
agent-browser click @e3
agent-browser screenshot output.png
agent-browser close
```

Use `http://localhost:<port>` for programmatic access, never the Coder proxy URLs.

## Temporary files

Screenshots and Claude-generated artifacts go in `.claude/screenshots/`. Never use `.run/` (reserved for dockge runtime) or the project root.

## Done means done

- ALL tests pass, not "all tests except the ones I decided are pre-existing"
- cargo clippy produces ZERO warnings
- If you believe a failure is unrelated to your changes, prove it by showing it
  fails the same way against the unmodified code. Do not assert this — demonstrate it.