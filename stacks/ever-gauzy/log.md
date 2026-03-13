# Ever Gauzy

## Source
- GitHub: https://github.com/ever-co/ever-gauzy
- Images: ghcr.io/ever-co/gauzy-api, ghcr.io/ever-co/gauzy-webapp

## Research
- Docker Compose demo file found at `docker-compose.demo.yml` in repo root
- Simplified from upstream demo compose (removed optional integrations: Sentry, PostHog, Jitsu, GitHub OAuth, etc.)
- Removed bind-mounted init script (`.deploy/db/init-user-db.sh`) - not needed for basic operation
- Removed `.env.demo.compose` env_file dependency - essential vars inlined with defaults
- Three services: PostgreSQL 17, Node.js API, Nginx webapp
- Default credentials: admin@ever.co / admin
- Web UI on port 4200, API on port 3000
