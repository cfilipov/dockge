# Tests Not Ported from Go

This documents Go tests from `handlers_test.go` that could not be ported to
the external WebSocket test suite, along with the reason for each.

## Server Internals (require in-process access)

| Go Test | Reason |
|---------|--------|
| `TestNeedSetup` | Dev mode auto-seeds admin user; can't test empty-DB state externally |
| `TestLoginRateLimitIntegration` | Requires injecting a custom `LoginRateLimiter` into the running server |
| `TestGlobalENVNotInBoltDB` | Requires direct BoltDB access to verify key absence |

## Server Mode (require prod-mode server)

| Go Test | Reason |
|---------|--------|
| `TestCORSRejectsWrongOriginInProdMode` | Tests run against `--dev` mode server |
| `TestCORSAcceptsAnyOriginInDevMode` | Trivially true in dev mode; no protocol assertion |

## Go-Specific Tests

| Go Test | Reason |
|---------|--------|
| `TestBinarySize` | Go build artifact test |
| `TestMemoryBudget` | Go runtime memory test |
| All `Benchmark*` tests | Go-specific performance benchmarks |

## Filesystem Verification

The following tests are ported for their **protocol-level assertions** (ack
`ok:true`), but their **filesystem verification** (checking files on disk)
could not be ported because the test suite runs externally and doesn't have
direct access to the server's filesystem:

| Go Test | What's Missing |
|---------|---------------|
| `TestSaveStack` | Verify `compose.yaml` written to disk |
| `TestSaveStackWithOverrideAndEnv` | Verify `.env` and `compose.override.yaml` on disk |
| `TestDeployStack` | Verify `compose.yaml` written to disk |
| `TestDeleteStackWithFiles` | Verify directory removed from disk |
| `TestForceDeleteStack` | Poll for directory removal |
| `TestGlobalENVRoundTrip` | Verify `global.env` file content |
| `TestGlobalENVDefaultDeletes` | Verify `global.env` deleted from disk |
| `TestGlobalENVEmptyDeletes` | Verify `global.env` deleted from disk |

## Container Lifecycle Tests

The following Go tests are ported but adapted because `SetStackRunning()` (which
calls the Docker API directly) is not available externally. Instead, these tests
issue `startStack` via WebSocket before testing stop/restart/etc:

- `TestStopStack`
- `TestRestartStack`
- `TestDownStack`
- `TestPauseAndResumeStack`
- `TestUpdateStack`
- All service tests (`TestStartService`, `TestStopService`, etc.)
- All docker resource tests (`TestDockerStats`, `TestContainerTop`, etc.)
- Terminal tests
- Broadcast tests
