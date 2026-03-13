# Accent

## Sources
- Repository: https://github.com/mirego/accent
- Docker Hub image: `mirego/accent`
- docker-compose.yml from repo (master branch): used as base, adapted for production
- README documents `docker run --env-file .env -p 4000:4000 mirego/accent`

## Changes from repo's docker-compose.yml
- Removed `version: '3.7'` (Compose V2)
- Replaced `build: .` with `image: mirego/accent` for production use
- Removed `container_name` and `network_mode: "host"` (not needed for compose)
- Added `DUMMY_LOGIN_ENABLED=true` for easy initial setup
- Added `POSTGRES_PASSWORD` and updated DATABASE_URL with password
- Added `restart: unless-stopped` to both services
- Removed host port mapping for PostgreSQL (not needed externally)
