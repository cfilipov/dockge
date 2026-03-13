# RecipeSage

- **Source**: https://github.com/julianpoy/RecipeSage
- **Description**: Self-hosted collaborative recipe manager with meal planning, shopping lists, and recipe auto-import
- **Architecture**: Node.js API, Angular frontend (static), PostgreSQL, Typesense search, Pushpin WebSocket proxy, Browserless for scraping
- **Images**: julianpoy/recipesage-selfhost-proxy, julianpoy/recipesage-selfhost (static + api tags), postgres:16, typesense/typesense, julianpoy/pushpin, ghcr.io/browserless/chromium
- **Compose reference**: Directly from julianpoy/recipesage-selfhost docker-compose.yml (v4.3.1 config, RecipeSage v3.0.10)
- **Notes**: 7-service stack. Optional ingredient-instruction-classifier container can be added for better import accuracy. Uses filesystem storage by default for self-hosted.
