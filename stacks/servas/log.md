# Servas
**Project:** https://github.com/beromir/Servas
**Source:** https://github.com/beromir/Servas
**Status:** done
**Compose source:** docker/compose.prod.yaml and docker/.env.prod.example from repo
## What was done
- Created compose.yaml from official production compose file
- Created .env.servas with Laravel app configuration (mounted into container)
- Created .env for compose variable substitution
- Used SQLite configuration (simpler, recommended option)
## Issues
- APP_KEY must be generated after first run via `docker exec -it servas php artisan key:generate --force`
