# Microweber Stack

## Research
- Found docker-compose.yml in microweber/microweber repo
- Original uses `build:` for php-apache service and bind-mounts source code
- Docker Hub has `microweber/microweber` image
- Services: PHP app with MariaDB

## Changes from upstream
- Replaced php-apache build service with `image: microweber/microweber:latest`
- Upgraded MariaDB from 10.3 to 10.6
- Removed bind-mount volumes, used named volumes
- Added environment variables for DB connection
