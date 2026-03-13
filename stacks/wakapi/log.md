# Wakapi

## Source
- GitHub: https://github.com/muety/wakapi
- Image: ghcr.io/muety/wakapi

## Research
- Docker Compose based on repo's `compose.yml` (replaced `build: .` with image reference)
- Simplified secrets to direct environment variables for ease of use
- Two services: Wakapi app + PostgreSQL 17
- Also supports SQLite (single container, no database service needed)
- Web UI on port 3000
- Compatible with WakaTime editor plugins
