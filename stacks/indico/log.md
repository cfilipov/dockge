# indico

## Sources
- Repository: https://github.com/indico/indico-containers (dedicated Docker setup repo)
- Compose file: https://github.com/indico/indico-containers/tree/master/indico-prod/docker-compose.yml
- Config files: indico.conf, logging.yaml, nginx.conf from same repo
- Docker image: getindico/indico (Docker Hub)

## Notes
- Original compose uses `build: worker` for the indico image; replaced with pre-built `getindico/indico:latest`
- Multi-service setup: web (Flask/uWSGI), celery worker, celery beat, Redis, PostgreSQL 15, Nginx reverse proxy
- Config files (indico.conf, logging.yaml, nginx.conf) are bind-mounted from the stack directory
- Environment variables for DB connection defined in .env file
- Nginx serves on port 8080 (configurable via NGINX_PORT), direct Flask access on port 9090
