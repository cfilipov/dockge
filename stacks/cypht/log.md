# Cypht Stack

## Sources
- Production docker-compose.yaml from official repo: https://github.com/cypht-org/cypht/blob/master/docker/docker-compose.yaml
- Docker Hub image: https://hub.docker.com/r/cypht/cypht

## Changes from upstream
- Changed host port from 80 to 8080 to avoid privileged port conflicts
- Changed image tag from `2.7.0` to `latest` for better maintainability
- Removed explicit port mapping on MariaDB (not needed for inter-service communication)

## Services
- **cypht**: Cypht webmail client (PHP + Nginx + Supervisor)
- **db**: MariaDB 10 database for session/user config storage

## Notes
- Default admin credentials: admin/admin (set via AUTH_USERNAME/AUTH_PASSWORD env vars)
- Uses MariaDB healthcheck to ensure DB is ready before Cypht starts
