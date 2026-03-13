# Revel

- **Source**: https://github.com/letsrevel/infra
- **Category**: Polls and Events
- **Description**: Event management platform with questionnaires, surveys, and participant management. Django-based with Celery for async tasks.
- **Services**: caddy (reverse proxy), web (Django/Gunicorn), celery_default (worker), celery_beat (scheduler), db (PostgreSQL), pgbouncer (connection pooler), redis
- **Notes**: Compose derived from upstream letsrevel/infra docker-compose.yml. PgBouncer used for connection pooling in transaction mode.
