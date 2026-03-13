# MyMangaDB

## Source
- GitHub: https://github.com/FabianRolfMatthiasNoll/MyMangaDB

## Research
- Found docker-compose.yml in repository - uses Traefik with build: directives
- Upstream compose uses build: context for both frontend and backend (no pre-built images in compose)
- Replaced build: with GHCR image references (ghcr.io/fabianrolfmatthiasnoll/mymangadb-*)
- Removed Traefik reverse proxy (not needed for standalone deployment)
- Simplified to direct port mapping

## Compose
- Images: ghcr.io/fabianrolfmatthiasnoll/mymangadb-backend:latest, ghcr.io/fabianrolfmatthiasnoll/mymangadb-frontend:latest
- Ports: 8080 (API), 3000 (Web UI)
- .env file for API_TOKEN variable substitution
- Named volume for database persistence
