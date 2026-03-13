# Mealie

- **Source**: https://github.com/mealie-recipes/mealie
- **Description**: Self-hosted recipe manager and meal planner with a clean UI and rich API
- **Architecture**: Python/FastAPI backend with Vue.js frontend, SQLite or PostgreSQL
- **Images**: ghcr.io/mealie-recipes/mealie
- **Compose reference**: Based on upstream docker/docker-compose.yml (simplified to SQLite single-container)
- **Notes**: Single container serves both API and frontend. Supports recipe scraping, meal planning, shopping lists, and multi-user households. Very popular project (16k+ stars).
