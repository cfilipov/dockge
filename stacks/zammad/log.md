# Zammad

## Source
- GitHub: https://github.com/zammad/zammad
- Docker Compose repo: https://github.com/zammad/zammad-docker-compose
- Compose: https://github.com/zammad/zammad-docker-compose/blob/master/docker-compose.yml

## Research
- Official docker-compose.yml from zammad-docker-compose repo
- Image: ghcr.io/zammad/zammad:7.0.0-9
- Multiple services: PostgreSQL, Redis, Elasticsearch, Memcached, Nginx, Rails server, Scheduler, WebSocket, Backup, Init
- Resolved env var substitutions to concrete defaults
- Web UI on port 8080 via Nginx
- DB credentials: zammad / zammad
