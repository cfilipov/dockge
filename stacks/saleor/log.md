# Saleor Stack

## Research
- Found docker-compose.yml in `saleor/saleor-platform` repo
- Images: `ghcr.io/saleor/saleor:3.22`, `ghcr.io/saleor/saleor-dashboard:latest`
- Uses PostgreSQL, Valkey (Redis-compatible), Celery worker, Mailpit, Jaeger
- Simplified: removed Jaeger (optional tracing), kept core services

## Compose
- Based on official saleor-platform docker-compose.yml
- API + Dashboard + Worker + PostgreSQL + Valkey + Mailpit
- Replaced env_file references with inline environment variables
