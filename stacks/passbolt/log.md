# Passbolt

## Sources
- GitHub repo: https://github.com/passbolt/passbolt_api
- Docker installation guide: https://www.passbolt.com/ce/docker
- Official compose download: https://download.passbolt.com/ce/docker/docker-compose-ce.yaml
- Docker Hub: https://hub.docker.com/r/passbolt/passbolt

## Notes
- Using the official Community Edition (CE) docker-compose downloaded from passbolt.com
- Two services: MariaDB 10.11 database + Passbolt CE application
- Uses wait-for.sh script to ensure DB is ready before starting the app
- Stores GPG keys and JWT tokens in named volumes
- Also available as non-root image (passbolt/passbolt:latest-ce-non-root)
