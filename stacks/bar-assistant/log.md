# Bar Assistant

- **Source**: https://github.com/karlomikus/bar-assistant
- **Description**: Self-hosted bar assistant / cocktail recipe manager with Meilisearch-powered search
- **Architecture**: PHP/Laravel backend (bar-assistant), Vue.js frontend (salt-rim), Meilisearch, Redis
- **Images**: barassistant/server, barassistant/salt-rim, getmeili/meilisearch, redis
- **Compose reference**: Based on upstream dev compose + production Docker Compose repo (bar-assistant/docker-compose)
- **Notes**: Salt Rim is the web frontend; the server image is the API. Meilisearch provides cocktail/ingredient search.
