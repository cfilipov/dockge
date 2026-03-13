# ONLYOFFICE Document Server

## Sources
- Official docker-compose.yml from Docker repo: https://github.com/ONLYOFFICE/Docker-DocumentServer/blob/master/docker-compose.yml
- Docker Hub image: `onlyoffice/documentserver`
- Main repo: https://github.com/ONLYOFFICE/DocumentServer (points to Docker-DocumentServer for deployment)

## Notes
- Compose file taken from the official Docker-DocumentServer repository
- Three services: Document Server, PostgreSQL 15, RabbitMQ 3
- Ports 80 and 443 exposed
- JWT enabled by default for security (was commented out in original; enabled here)
- Anonymous volumes from original converted to named volumes for persistence
- Health checks included for all three services
- Community Edition is free for up to 20 users
