# Saltcorn — Research Log

## Sources checked
1. `deploy/examples/test/docker-compose.yml` in repo — found official compose file
2. README.md — references `saltcorn/saltcorn` Docker image and test compose example

## Compose file origin
From the official repository at `deploy/examples/test/docker-compose.yml`.

## Modifications
- Removed `version: "3.7"` field (Compose V2 format)
- Removed traefik labels (not needed for basic setup)
- Removed init SQL volume mount (optional)
- Added .env file with placeholder values for required environment variables
