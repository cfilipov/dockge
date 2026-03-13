# Featbit

## Source
- Repository: https://github.com/featbit/featbit
- Compose file: https://raw.githubusercontent.com/featbit/featbit/main/docker-compose.yml

## Notes
- Official docker-compose.yml from the repository root
- Removed the `container_name` from postgresql service for Compose V2 compatibility
- Removed the init script volume mount (`infra/postgresql/docker-entrypoint-initdb.d/`) since it references repo-local files; the app handles schema creation on startup
- Removed custom subnet IPAM config (not needed for basic operation)
- Services: UI (port 8081), API server (port 5000), evaluation server (port 5100), data analytics server (port 8200), PostgreSQL (port 5432)
