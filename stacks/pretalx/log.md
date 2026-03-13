# pretalx

## Sources
- Repository: https://github.com/pretalx/pretalx
- Docker repository: https://github.com/pretalx/pretalx-docker
- Compose file: https://github.com/pretalx/pretalx-docker/blob/main/docker-compose.yml
- Config file: https://github.com/pretalx/pretalx-docker/blob/main/conf/pretalx.cfg
- Docker image: pretalx/standalone (Docker Hub)

## Notes
- Community-maintained Docker setup (not officially supported per pretalx docs)
- Three services: pretalx app (Gunicorn), PostgreSQL 15, Redis
- Config file (pretalx.cfg) bind-mounted from ./conf/
- App accessible on port 80
- Uses pretalx/standalone:latest pre-built image
- Database password in pretalx.cfg must match POSTGRES_PASSWORD in compose
