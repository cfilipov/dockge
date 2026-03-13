# ManageMeals

- **Source**: https://github.com/managemeals
- **Description**: Self-hosted recipe manager with web scraping, search, and multi-platform clients
- **Architecture**: Node.js API, SvelteKit web frontend, MongoDB, Valkey (Redis), Typesense search, recipe scraper
- **Images**: ghcr.io/managemeals/manage-meals-api, ghcr.io/managemeals/manage-meals-web, ghcr.io/managemeals/manage-meals-scraper, ghcr.io/managemeals/manage-meals-search-sync, mongo:7, valkey/valkey:8, typesense/typesense:28.0
- **Compose reference**: Based on upstream docker-compose.selfhost.yaml (removed mongo-express admin UI)
- **Notes**: 7-service stack. Has companion Firefox extension and mobile apps. Typesense provides fuzzy recipe search.
