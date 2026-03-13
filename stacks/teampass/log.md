# Teampass

## Sources
- GitHub repo: https://github.com/nilsteampassnet/TeamPass
- Docker Compose file: https://github.com/nilsteampassnet/TeamPass/blob/master/docker-compose.yml

## Notes
- Compose taken from the official TeamPass repository
- Simplified from the original: removed nginx reverse proxy and multi-network setup for standalone use
- Uses community-maintained Docker image (dormancygrace/teampass)
- Two services: MariaDB database + TeamPass PHP application
- Original compose used bind mounts; switched to named volumes for portability
- Replace default passwords before production use
