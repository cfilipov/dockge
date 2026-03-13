# Listmonk Stack

## Sources
- Repository: https://github.com/knadh/listmonk
- Compose file: https://raw.githubusercontent.com/knadh/listmonk/master/docker-compose.yml

## Notes
Listmonk is a high-performance, self-hosted newsletter and mailing list manager.
Compose file taken directly from the official repository with no modifications.
The app command runs install (idempotent), upgrade, then starts the server.

## Services
- **app** (`listmonk/listmonk:latest`) - Newsletter/mailing list manager (port 9000)
- **db** (`postgres:17-alpine`) - PostgreSQL database
