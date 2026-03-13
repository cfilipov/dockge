# Warpgate

## Sources
- Official repo: https://github.com/warp-tech/warpgate
- Official docker-compose.yml: https://raw.githubusercontent.com/warp-tech/warpgate/main/docker/docker-compose.yml
- Docker setup docs: https://warpgate.null.page/getting-started-on-docker/

## Notes
- Warpgate is a smart SSH, HTTPS, and MySQL bastion host for Linux
- Official image: `ghcr.io/warp-tech/warpgate`
- Compose taken from the official docker/docker-compose.yml in the repository
- Added restart policy and container_name
- Before first run, initialize config: `docker compose run warpgate setup`
- Port 2222: SSH access
- Port 8888: HTTPS web admin interface
- Port 33306: MySQL protocol proxy
- Data persisted in ./data bind mount
