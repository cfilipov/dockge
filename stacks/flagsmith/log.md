# Flagsmith

## Source
- Repository: https://github.com/Flagsmith/flagsmith
- Compose file: https://raw.githubusercontent.com/Flagsmith/flagsmith/main/docker-compose.yml

## Notes
- Official docker-compose.yml from the repository root
- Removed `container_name` from postgres service for Compose V2 compatibility
- Services: PostgreSQL (internal), Flagsmith web app (port 8000), task processor (port 8001)
- Uses custom registry `docker.flagsmith.com` for the Flagsmith image
- PostgreSQL healthcheck ensures DB is ready before Flagsmith starts
