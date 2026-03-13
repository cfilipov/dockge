# Tolgee

## Sources
- Repository: https://github.com/tolgee/tolgee-platform
- Docker Hub image: `tolgee/tolgee`
- Official docs: https://docs.tolgee.io/platform/self_hosting/running_with_docker
- docker-compose.yml from repo's docker/ directory (development compose)
- Official docs provide both simple and external PostgreSQL compose examples

## Compose derivation
- Based on the "With External PostgreSQL Database" example from official docs
- Removed config.yaml volume mount (not needed for basic setup)
- Configured spring.datasource.* environment variables directly instead
- Set `tolgee.postgres-autostart.enabled=false` since external Postgres is used
- Used named volumes instead of bind mounts
- Added `restart: unless-stopped`
- Removed unnecessary port 25432 exposure
- Upgraded Postgres from 13 to 16
