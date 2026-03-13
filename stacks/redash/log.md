# Redash
**Project:** https://github.com/getredash/redash
**Source:** https://github.com/getredash/redash
**Status:** done
**Compose source:** compose.yaml from repository, adapted for production use

## What was done
- Created compose.yaml based on official compose.yaml
- Replaced build: with redash/redash:10.1.0.b50633 image
- Includes server, scheduler, worker, PostgreSQL, and Redis
- Created .env with database password and secret key

## Issues
- None
