# Xandikos

## Sources
- GitHub repo: https://github.com/jelmer/xandikos
- Docker Compose: examples/docker-compose.yml in repo
- Container registry: ghcr.io/jelmer/xandikos

## Notes
- Compose taken directly from examples/docker-compose.yml in the repo
- Removed `version: "3.4"` for Compose V2 compatibility
- Changed host path bind mount to named volume for portability
- Port 8000 for CalDAV/CardDAV, port 8001 for metrics
- Lightweight Python CalDAV/CardDAV server with Git backend
- Includes healthcheck from the official example
