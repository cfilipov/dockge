# TuxSEO

## Sources
- Repository: https://github.com/rasulkireev/TuxSEO
- Compose file (prod): https://github.com/rasulkireev/TuxSEO/blob/main/docker-compose-prod.yml
- Compose file (local): https://github.com/rasulkireev/TuxSEO/blob/main/docker-compose-local.yml

## Notes
- Based on the production docker-compose-prod.yml
- Uses pre-built GHCR images for backend and workers
- Replaced env_file references with inline environment variables using defaults
- Includes PostgreSQL (custom image), Redis, backend (Django), and background workers
- Access at http://localhost:8000
