# Superset
**Project:** https://github.com/apache/superset
**Source:** https://github.com/apache/superset
**Status:** done
**Compose source:** docker-compose-image-tag.yml from repository

## What was done
- Created compose.yaml based on official docker-compose-image-tag.yml
- Uses apache/superset:latest image
- Includes PostgreSQL 17, Redis 7, init container, worker, and beat scheduler
- Created .env with database password and secret key

## Issues
- None
