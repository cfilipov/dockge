# Postorius Stack

## Sources
- Project: https://gitlab.com/mailman/postorius/
- Docker setup: https://github.com/maxking/docker-mailman
- Compose file: https://raw.githubusercontent.com/maxking/docker-mailman/main/docker-compose.yaml

## Notes
Postorius is the web-based list management interface for GNU Mailman 3.
It does not have a standalone Docker image — it is bundled in the `maxking/mailman-web:0.4`
image alongside HyperKitty (the archiver). This is the same compose setup used for
HyperKitty and Mailman stacks, from the official docker-mailman repository.

## Services
- **mailman-core** (`maxking/mailman-core:0.4`) - Mailman 3 core with REST API (port 8001) and LMTP (port 8024)
- **mailman-web** (`maxking/mailman-web:0.4`) - Django app serving Postorius + HyperKitty (port 8000/8080)
- **database** (`postgres:12-alpine`) - PostgreSQL database
