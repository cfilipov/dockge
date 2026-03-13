# Nextcloud Stack

## Source
- Docker Hub: nextcloud:latest
- Docs: https://hub.docker.com/_/nextcloud

## Services
- **nextcloud**: Nextcloud server on port 8080
- **db**: MariaDB 11 database
- **redis**: Redis for file locking and caching

## Notes
- Official Docker Hub image
- MariaDB configured with READ-COMMITTED isolation and binary logging (recommended)
- Redis used for file locking performance
