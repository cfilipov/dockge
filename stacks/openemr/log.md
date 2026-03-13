# OpenEMR

## Sources
- Repository: https://github.com/openemr/openemr
- Compose file: `docker/production/docker-compose.yml` from master branch
- Documentation: `DOCKER_README.md`

## Notes
- Image: `openemr/openemr:latest` (Docker Hub)
- Database: MariaDB 11.8
- Default credentials: admin/pass (OE_USER/OE_PASS)
- Exposes ports 80 (HTTP) and 443 (HTTPS)
- Uses named volumes for database, logs, and site data
- Taken directly from official production compose file
