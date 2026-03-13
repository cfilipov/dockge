# Bencher

## Source
- Repository: https://github.com/bencherdev/bencher
- Compose file: https://raw.githubusercontent.com/bencherdev/bencher/main/docker-compose.yml

## Notes
- Official docker-compose.yml found in repo root
- Two services: API (port 61016) and Console (port 3000)
- Images from GHCR: ghcr.io/bencherdev/bencher-api, ghcr.io/bencherdev/bencher-console
- Adapted bind mounts to named volumes for portability
- Changed INTERNAL_API_URL to use Docker service name instead of host.docker.internal
