# kener

## Sources
- Official repo: https://github.com/rajnandan1/kener
- Official docker-compose.yml in repo root
- README Docker instructions

## Notes
- kener is a modern status page application (v4)
- Official images: `rajnandan1/kener:latest` (Docker Hub) and `ghcr.io/rajnandan1/kener:latest` (GHCR)
- Requires Redis for BullMQ queues, caching, and scheduler
- Default database is SQLite (stored in volume); PostgreSQL and MySQL also supported
- KENER_SECRET_KEY and ORIGIN are required environment variables
- Compose based on the official docker-compose.yml from the repository
