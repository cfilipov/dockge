# Overleaf

- **Source**: https://github.com/overleaf/overleaf
- **Category**: Note-taking & Editors
- **Description**: Self-hosted collaborative LaTeX editor (Community Edition)
- **Services**: sharelatex (Overleaf app), mongo (MongoDB 6.0 with replica set), redis (Redis 6.2)
- **Ports**: 8088 (web UI)
- **Based on**: Official docker-compose.yml from repository (simplified, removed Server Pro options)
- **Config files**: mongodb-init-replica-set.js (initializes MongoDB replica set required by Overleaf)
