# Lago

- **Source**: https://github.com/getlago/lago
- **Compose reference**: `docker-compose.yml` from official repo (main branch)
- **Status**: ok
- **Services**: db (postgres-partman), redis, migrate, api, front, api-worker, api-clock, pdf (gotenberg)
- **Notes**: Based on official compose. Simplified the YAML anchor/alias pattern into explicit environment blocks for Dockge compatibility. Pinned to v1.43.0. Removed commented-out optional workers (events, pdfs, billing, webhook, analytics, ai-agent) and SSL/certbot configuration. All encryption keys need to be changed for production use.
