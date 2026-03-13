# Miniflux

## Source
https://github.com/miniflux/v2

## Notes
- Minimalist and opinionated feed reader written in Go
- Official Docker image: miniflux/miniflux
- Compose based on upstream contrib/docker-compose/basic.yml
- Requires PostgreSQL; runs migrations on startup (RUN_MIGRATIONS=1)
- Creates admin user on first run (CREATE_ADMIN=1)
- Fast, lightweight, keyboard-driven UI
