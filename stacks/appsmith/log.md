# Appsmith — Research Log

## Sources checked
1. `deploy/docker/docker-compose.yml` in repo (release branch) — found official compose file

## Compose file origin
From the official repository at `deploy/docker/docker-compose.yml` on the `release` branch.

## Modifications
- Removed `version: "3"` field (Compose V2 format)
- Removed `index.docker.io/` prefix from image name (unnecessary)
- Added `restart: unless-stopped`
