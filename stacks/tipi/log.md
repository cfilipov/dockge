# Tipi (Runtipi)

## Source
- GitHub: https://github.com/runtipi/runtipi
- Reference: docker-compose.prod.yml from develop branch

## Description
Runtipi is a self-hosted homeserver management platform. It provides a web dashboard for installing and managing self-hosted applications via a Traefik reverse proxy, with PostgreSQL for data storage and RabbitMQ for task queuing.

## Services
- **runtipi** - Main application server (Node.js/Next.js dashboard)
- **runtipi-reverse-proxy** - Traefik v3.5 reverse proxy for routing
- **runtipi-db** - PostgreSQL 14 database
- **runtipi-queue** - RabbitMQ 4 message queue

## Ports
- 80 (HTTP via Traefik)
- 443 (HTTPS via Traefik)
- 8080 (Traefik dashboard)

## Notes
- The official compose uses `build:` context; replaced with ghcr.io image for fixture
- Traefik requires Docker socket access for service discovery
- Health checks configured on both database and main app
- Volumes mapped for media, state, repos, apps, and logs
