# Shelf

## Source
- Repository: https://github.com/Shelf-nu/shelf.nu
- Docker image: ghcr.io/shelf-nu/shelf.nu:latest
- Documentation: https://docs.shelf.nu/docker

## Compose Source
- Converted from official docker run command in docs: https://docs.shelf.nu/docker
- No official docker-compose.yml exists in the repository

## Notes
- Requires an external Supabase instance (database not included in compose)
- All Supabase and SMTP env vars must be configured before use
- Database migrations must be run separately before first start

## Services
- **shelf**: Asset management web app (port 3000 -> 8080)
