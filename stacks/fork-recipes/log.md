# Fork Recipes

- **Source**: https://github.com/mikebgrep/fork.recipes
- **Description**: Self-hosted recipe manager with separate API backend and web frontend
- **Architecture**: Django backend (forkapi), Django frontend (forkrecipes), Nginx reverse proxy, SQLite storage
- **Images**: mikebgrep/forkapi, mikebgrep/forkrecipes, nginx:alpine
- **Compose reference**: Based on upstream docker-compose-postgres.yml (simplified to sqlite variant)
- **Notes**: Uses uWSGI sockets shared via volumes. Nginx proxies to both backend and frontend via Unix sockets.
