# Ziit

## Source
- GitHub: https://github.com/0pandadev/ziit
- Image: ghcr.io/0pandadev/ziit

## Research
- Docker Compose from `docker-compose.yml` in repo root
- Two services: Ziit app (Nuxt) + TimescaleDB (PostgreSQL 17)
- Removed optional OAuth config (GitHub, Epilogue) for simplicity
- Removed `post_start` hook (chown) as it may not be supported in all compose versions
- NUXT_PASETO_KEY and NUXT_ADMIN_KEY must be generated before first run
- Changed image tag from pinned v1.1.1 to latest
- Web UI on port 3000
