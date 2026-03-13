# HyperKitty Stack

## Sources
- Repository: https://gitlab.com/mailman/hyperkitty
- Docker setup: https://github.com/maxking/docker-mailman
- Compose file: https://raw.githubusercontent.com/maxking/docker-mailman/main/docker-compose.yaml

## Notes
HyperKitty is the archiver web interface for GNU Mailman 3. It does not have a standalone Docker image.
It is bundled in the `maxking/mailman-web:0.4` image alongside Postorius (the list management web UI).
The compose file is from the official docker-mailman repository which provides the full Mailman 3 suite.

## Services
- **mailman-core** (`maxking/mailman-core:0.4`) - Mailman 3 core with REST API (port 8001) and LMTP (port 8024)
- **mailman-web** (`maxking/mailman-web:0.4`) - Django app serving HyperKitty + Postorius (port 8000/8080)
- **database** (`postgres:12-alpine`) - PostgreSQL database
