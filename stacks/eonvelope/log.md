# Eonvelope
**Project:** https://gitlab.com/dacid99/eonvelope
**Source:** https://gitlab.com/dacid99/eonvelope
**Status:** done
**Compose source:** docker/docker-compose.yml from GitLab repo (master branch)
## What was done
- Created compose.yaml based on official docker-compose.yml
- Includes db (MariaDB) and web (dacid99/eonvelope) services
- Created .env with essential variables (subset of the 40+ available)
- Created mount point directories (archive, log, mysql)
## Issues
- Many optional environment variables omitted for brevity; only essential ones included
