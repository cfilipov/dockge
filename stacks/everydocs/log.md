# EveryDocs

## What was done
- Based on official docker-compose.yaml from jonashellmann/everydocs-core
- Services: everydocs_core (API), everydocs_web (frontend), MariaDB
- Images: jonashellmann/everydocs:latest, jonashellmann/everydocs-web:latest, mariadb:10.11
- Ports: 5678 (API), 8080 (web UI)
- Removed bind mount for web config (not needed for mock testing)
- Secrets extracted to .env
