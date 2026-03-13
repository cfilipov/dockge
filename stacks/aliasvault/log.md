# AliasVault

## Sources
- GitHub repo: https://github.com/aliasvault/aliasvault
- Docker Compose file: https://github.com/aliasvault/aliasvault/blob/main/docker-compose.yml
- Container registry: ghcr.io/aliasvault

## Notes
- Full-featured email alias and password management platform
- 7 services: postgres, client, api, admin, reverse-proxy, smtp, task-runner
- All images hosted on GitHub Container Registry (ghcr.io/aliasvault/*)
- Original compose uses env_file (.env) for configuration; removed for simplicity
- Original uses variable substitution for ports (HTTP_PORT, HTTPS_PORT, SMTP_PORT, SMTP_TLS_PORT); replaced with defaults
- Requires secrets directory with postgres_password file for database authentication
