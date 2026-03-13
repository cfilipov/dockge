# Pretix Stack

## Research
- Docker Hub image: `pretix/standalone` (tags: latest, stable, 2026.2.0)
- Pretix requires PostgreSQL and Redis
- Configuration via PRETIX_* environment variables

## Compose
- Used `pretix/standalone:stable` image
- PostgreSQL 16 for database
- Redis 7 for caching and Celery broker
- Environment variables for database, Redis, and URL configuration
