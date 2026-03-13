# wger

## Sources
- Repository: https://github.com/wger-project/docker
- Compose file: `docker-compose.yml` from master branch
- Config files: `config/prod.env`, `config/nginx.conf` from master branch

## Notes
- Image: `docker.io/wger/server:latest` (Docker Hub)
- 6 services: web app, nginx, PostgreSQL, Redis, celery worker, celery beat
- Nginx required for static file serving (per project docs)
- Removed redis.conf bind-mount (default redis config is sufficient)
- Config files stored in `config/` subdirectory matching upstream layout
