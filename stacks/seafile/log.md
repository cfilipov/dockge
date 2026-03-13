# Seafile

## Source
- GitHub: https://github.com/haiwen/seafile
- Docker Hub: https://hub.docker.com/r/seafileltd/seafile-mc

## Description
Seafile is an open source cloud storage system with privacy protection and teamwork features. It supports file syncing, sharing, and collaboration with client apps for all major platforms.

## Stack Components
- **seafile**: Seafile server (seafileltd/seafile-mc:13.0-latest)
- **mariadb**: MariaDB database backend (mariadb:11)
- **memcached**: Memcached for caching (memcached:1.6-alpine)

## Ports
- 8082: Seafile web interface (mapped from container port 80)

## Volumes
- seafile_data: Seafile data and configuration
- mariadb_data: Database storage

## Configuration Notes
- Admin account is created on first run via SEAFILE_ADMIN_EMAIL/PASSWORD
- SEAFILE_SERVER_HOSTNAME should match the public-facing domain
- Memcached configured with 256MB memory limit
- MariaDB root password shared between seafile and mariadb services
