# Rallly

## Sources
- GitHub: https://github.com/lukevella/rallly
- Self-hosted repo: https://github.com/lukevella/rallly-selfhosted
- Docker Compose: https://raw.githubusercontent.com/lukevella/rallly-selfhosted/main/docker-compose.yml
- Docker Hub: https://hub.docker.com/r/lukevella/rallly
- Self-hosting docs: https://support.rallly.co/self-hosting

## Notes
- Compose taken from official self-hosted repo (rallly-selfhosted)
- Image `lukevella/rallly:latest` is the official Docker Hub image
- Added `condition: service_healthy` to depends_on for proper startup ordering
- Inlined essential env vars from config.env into compose (DATABASE_URL, SECRET_PASSWORD, etc.)
- SECRET_PASSWORD must be at least 32 characters
- SMTP config omitted (optional, needed for email features)
- PostgreSQL 14.2 as specified in official compose
