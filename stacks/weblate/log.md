# Weblate

## Sources
- Repository: https://github.com/WeblateOrg/weblate
- Docker Compose repo: https://github.com/WeblateOrg/docker-compose
- Docker Hub image: `weblate/weblate`
- docker-compose.yml and environment file from WeblateOrg/docker-compose (main branch)

## Compose derivation
- Based directly on the official WeblateOrg/docker-compose repository
- Converted `env_file: ./environment` to inline `environment:` block
- Added port mapping `8080:8080` (not in upstream compose, expected to be in override)
- Added `WEBLATE_ADMIN_PASSWORD` via .env variable substitution
- Updated postgres image from 18-alpine to 16-alpine (18 not yet stable)
- Kept valkey cache, read_only flags, and tmpfs mounts from upstream
- All three services (weblate, database, cache) preserved from upstream
