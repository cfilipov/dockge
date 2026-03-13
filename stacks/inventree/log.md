# InvenTree

## Source
- Repository: https://github.com/inventree/InvenTree
- Docker image: inventree/inventree:stable

## Compose Source
- Fetched from: https://raw.githubusercontent.com/inventree/InvenTree/master/contrib/container/docker-compose.yml
- Official production docker-compose from contrib/container/
- .env based on: https://raw.githubusercontent.com/inventree/InvenTree/master/contrib/container/.env
- Caddyfile based on: https://raw.githubusercontent.com/inventree/InvenTree/master/contrib/container/Caddyfile

## Changes from Original
- Removed `version` field (Compose V2)
- Replaced INVENTREE_EXT_VOLUME bind mount with named volumes
- Made DB credentials use defaults instead of required error syntax
- Changed default HTTP/HTTPS ports to 1080/1443 to avoid privileged ports
- Simplified Caddyfile static/media paths to match named volume mounts

## Services
- **inventree-db**: PostgreSQL 17 database
- **inventree-cache**: Redis 7 cache
- **inventree-server**: InvenTree web server (gunicorn)
- **inventree-worker**: Background task worker
- **inventree-proxy**: Caddy reverse proxy (ports 1080/1443)
