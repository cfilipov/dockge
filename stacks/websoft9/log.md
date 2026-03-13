# Websoft9

## Source
- GitHub: https://github.com/Websoft9/websoft9
- Reference: docker/docker-compose.yml and docker/.env from main branch

## Description
Websoft9 is a self-hosting and DevOps platform for running open-source applications. It provides a web console (based on Nginx Proxy Manager) for managing multiple apps, with Portainer for Docker management, Gitea for app repository storage, and a custom app hub for discovery and installation.

## Services
- **apphub** - Application hub and management dashboard
- **deployment** - Portainer-based container deployment engine
- **git** - Gitea instance for storing app configurations
- **proxy** - Nginx-based reverse proxy with Let's Encrypt and ModSecurity

## Ports
- 80 (HTTP gateway)
- 443 (HTTPS gateway)
- 9000 (Management console)

## Notes
- Requires an external Docker network named "websoft9" (created outside compose)
- All services require Docker socket access for container management
- Version-pinned images from websoft9dev Docker Hub organization
- Compose and .env taken directly from official repository
