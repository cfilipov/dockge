# Paperless-ngx

## What was done
- Based on official docker-compose.postgres.yml from paperless-ngx/paperless-ngx
- Services: webserver (Paperless-ngx), PostgreSQL, Redis (broker)
- Images: ghcr.io/paperless-ngx/paperless-ngx:latest, postgres:16, redis:8
- Port: 8000 (web UI)
- Added common env vars inline (no .env needed)
- Consume and export directories as bind mounts
