# Etherpad

## Sources
- Official docker-compose.yml from repo: https://github.com/ether/etherpad-lite/blob/develop/docker-compose.yml
- Docker Hub image: `etherpad/etherpad`

## Notes
- Compose file taken directly from the official repository (develop branch)
- Uses PostgreSQL 15 as the database backend
- Port 9001 exposed for the web interface
- Admin password configurable via environment variable
- Includes plugin and var data volumes for persistence
