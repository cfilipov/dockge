# TeamMapper

## Sources
- Official repo: https://github.com/b310-digital/teammapper
- Production compose: https://github.com/b310-digital/teammapper/blob/main/docker-compose-prod.yml
- GHCR image: https://github.com/b310-digital/teammapper/pkgs/container/teammapper

## Notes
- TeamMapper is a collaborative mind mapping tool
- Official image: `ghcr.io/b310-digital/teammapper:latest`
- Compose adapted from docker-compose-prod.yml (replaced `build:` with published GHCR image)
- Requires PostgreSQL 15
- App listens on port 3000, mapped to host port 80 by default
- SSL disabled for local PostgreSQL (production compose had SSL config for external DB)
- DELETE_AFTER_DAYS controls automatic cleanup of old maps (default 30 days)
