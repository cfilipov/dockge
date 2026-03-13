# MyFin Budget

- **Source**: https://github.com/afaneca/myfin
- **Compose reference**: `docker-compose.yml` from official repo (master branch)
- **Status**: ok
- **Services**: db (MySQL 8.4), myfin-api (backend), myfin-frontend (web UI)
- **Notes**: Based directly on the official docker-compose.yml. Both app images hosted on GHCR. The frontend needs `VITE_MYFIN_BASE_API_URL` set to the API's public URL. Removed `service_healthy` condition from frontend depends_on (API healthcheck not defined in original compose).
