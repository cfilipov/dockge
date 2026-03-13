# monetr

- **Source**: https://github.com/monetr/monetr
- **Compose reference**: `docker-compose.yaml` from official repo (main branch)
- **Status**: ok
- **Services**: postgres (17), valkey (Redis-compatible cache), monetr (app)
- **Notes**: Based directly on the official docker-compose.yaml. Uses Valkey instead of Redis. The monetr container auto-migrates and generates TLS certificates on startup. Image hosted on GHCR.
