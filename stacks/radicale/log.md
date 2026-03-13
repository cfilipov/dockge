# Radicale

## Sources
- GitHub repo: https://github.com/Kozea/Radicale
- Docker Compose: compose.yaml in repo root (master branch)
- Container registry: ghcr.io/kozea/radicale

## Notes
- Official compose.yaml taken directly from the Radicale repository
- Simplified volume definitions (removed bind mount driver_opts for portability)
- Removed `name:` top-level field for Compose V2 compatibility
- Runs on port 5232
- Lightweight CalDAV/CardDAV server
