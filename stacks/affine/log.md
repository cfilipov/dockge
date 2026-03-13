# AFFiNE Community Edition

## Sources
- Official repo: https://github.com/toeverything/AFFiNE
- Official self-host compose: https://github.com/toeverything/AFFiNE/blob/canary/.docker/selfhost/compose.yml
- Official .env.example: https://github.com/toeverything/AFFiNE/blob/canary/.docker/selfhost/.env.example

## Notes
- AFFiNE is a privacy-focused, local-first, open-source workspace (alternative to Notion)
- Official image: `ghcr.io/toeverything/affine:stable`
- Compose taken directly from the official `.docker/selfhost/compose.yml`
- Requires PostgreSQL (with pgvector) and Redis
- Migration job runs before the main service starts
- Web UI accessible on port 3010
- .env file provides variable substitution for DB credentials, storage paths, and revision
