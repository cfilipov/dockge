# Taiga

- **Status**: created
- **Source**: https://github.com/taigaio/taiga-docker
- **Images**: taigaio/taiga-back, taigaio/taiga-front, taigaio/taiga-events, taigaio/taiga-protected
- **Description**: Open-source agile project management platform
- **Notes**: Based on the official taiga-docker repository compose file. Includes 9 services: PostgreSQL, backend, async worker, frontend, events, two RabbitMQ instances, protected media handler, and nginx gateway. Nginx config (taiga.conf) is bind-mounted.
