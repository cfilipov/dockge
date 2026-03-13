# OpenSign

## What was done
- Based on official docker-compose.yml from opensignlabs/opensign
- Removed Caddy reverse proxy (not needed for local testing)
- Services: server (Parse/Express API), client (React frontend), MongoDB
- Images: opensign/opensignserver:main, opensign/opensign:main, mongo:7
- Ports: 8080 (API), 3000 (web UI)
- Credentials extracted to .env
