# Vaultwarden

## Sources
- GitHub repo: https://github.com/dani-garcia/vaultwarden
- Docker Compose wiki: https://github.com/dani-garcia/vaultwarden/wiki/Using-Docker-Compose
- Docker Hub: https://hub.docker.com/r/vaultwarden/server

## Notes
- Using the minimal template from the official wiki (no reverse proxy)
- Single service: vaultwarden/server:latest
- Uses SQLite by default (no external database required)
- Wiki also provides examples with Caddy reverse proxy for HTTPS
- Changed bind mount to named volume for portability
- Set DOMAIN environment variable for production use with HTTPS
