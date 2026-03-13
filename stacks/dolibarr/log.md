# Dolibarr Stack Research Log

## Sources
- Main repo: https://github.com/Dolibarr/dolibarr (points to separate docker repo)
- Docker repo: https://github.com/Dolibarr/dolibarr-docker
- Docker Hub: https://hub.docker.com/r/dolibarr/dolibarr
- Reference compose: https://raw.githubusercontent.com/Dolibarr/dolibarr-docker/main/docker-compose.yml

## Notes
- Official docker-compose.yml in dolibarr-docker uses build contexts and Docker secrets files, simplified for self-hosted use
- Image: `dolibarr/dolibarr:latest`
- Database: MariaDB (also supports PostgreSQL)
- Key env vars: DOLI_DB_HOST, DOLI_DB_NAME, DOLI_DB_USER, DOLI_DB_PASSWORD, DOLI_ADMIN_LOGIN, DOLI_ADMIN_PASSWORD
- DOLI_INSTALL_AUTO=1 enables auto-installation on first boot
- DOLI_INIT_DEMO=1 loads demo data
