# ToolJet — Research Log

## Sources checked
1. `deploy/docker/docker-compose.yaml` in repo — found production compose with `tooljet/tooljet-ce:latest`
2. `deploy/docker/.env.internal.example` — found environment variable template
3. Root `docker-compose.yaml` — dev-only (builds from source)

## Compose file origin
Based on the official production compose at `deploy/docker/docker-compose.yaml` with database service added from the dev compose configuration.

## Modifications
- Removed `version: "3"` field (Compose V2 format)
- Added PostgreSQL service (the production compose assumes external DB)
- Inlined essential environment variables with substitution from .env
- Removed PostgREST service (commented out / optional in original)
- Added .env file with placeholder values
