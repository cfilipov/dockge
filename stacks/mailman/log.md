# Mailman Stack

## Sources
- Project: https://gitlab.com/mailman/
- Docker setup: https://github.com/maxking/docker-mailman
- Compose file: https://raw.githubusercontent.com/maxking/docker-mailman/main/docker-compose.yaml

## Notes
GNU Mailman 3 is a mailing list management system. The official Docker deployment
uses the docker-mailman project which provides mailman-core and mailman-web images.
This is the same compose setup used for HyperKitty and Postorius (which are bundled
in the mailman-web image).

## Services
- **mailman-core** (`maxking/mailman-core:0.4`) - Mailman 3 core with REST API (port 8001) and LMTP (port 8024)
- **mailman-web** (`maxking/mailman-web:0.4`) - Django app serving Postorius + HyperKitty (port 8000/8080)
- **database** (`postgres:12-alpine`) - PostgreSQL database
