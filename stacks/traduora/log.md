# Traduora

## Sources
- Repository: https://github.com/ever-co/ever-traduora
- Docker Hub image: `everco/ever-traduora`
- docker-compose.yaml from repo (develop branch): default SQLite setup
- docker-compose.postgres.yaml also available in repo for PostgreSQL variant

## Compose derivation
- Based on the repo's docker-compose.yaml (SQLite variant, simplest setup)
- Removed `version: '3.7'` (Compose V2)
- Removed `build:` section (using pre-built image only)
- Removed `container_name` (let Compose manage names)
- Removed `networks:` block (default network is sufficient)
- Changed bind mount to named volume for data persistence
- Set NODE_ENV to production
- Changed `restart: on-failure` to `restart: unless-stopped`
