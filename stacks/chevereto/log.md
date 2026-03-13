# Chevereto

- **Source**: https://github.com/chevereto/chevereto
- **Docker image**: chevereto/chevereto:latest
- **Compose ref**: https://github.com/chevereto/docker (4.4 branch, docker-compose.yml.dist)
- **Description**: Self-hosted image hosting solution with albums, user management, and sharing
- **Services**: php (app), database (MariaDB)
- **Notes**: Simplified from upstream — removed nginx-proxy/letsencrypt references, exposed port directly. License key env var omitted (free tier).
