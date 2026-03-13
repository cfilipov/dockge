# Rybbit

- **Source**: https://github.com/rybbit-io/rybbit
- **Status**: ok
- **Images**: ghcr.io/rybbit-io/rybbit-backend, ghcr.io/rybbit-io/rybbit-client, clickhouse/clickhouse-server:25.4.2, postgres:17.4
- **Notes**: Based on upstream docker-compose.yml (master branch). Removed build directives and caddy profile. ClickHouse configs use compose `configs:` feature (inline content). Ports simplified from host-bound to plain mappings.
