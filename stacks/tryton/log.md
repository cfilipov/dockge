# Tryton Stack Research Log

## Sources
- Docker Hub: https://hub.docker.com/r/tryton/tryton
- Docker repo: https://foss.heptapod.net/tryton/tryton-docker
- Reference compose: https://foss.heptapod.net/tryton/tryton-docker/-/raw/branch/default/compose.yml

## Notes
- Image: `tryton/tryton:latest` (also supports `-office` suffix for document conversion)
- Database: PostgreSQL
- Port: 8000
- Three services: server (web + admin init), cron (scheduled tasks), postgres
- Server command waits for DB, initializes admin user, then starts app server
- Default admin password: "admin" (set via PASSWORD env var)
- Env vars: VERSION, PASSWORD, EMAIL, PG_VERSION, DB_NAME, DB_PASSWORD
- Changed cron service to connect to "server" instead of "tryton" (service name)
- Access at http://localhost:8000/
