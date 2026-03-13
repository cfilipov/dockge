# Litlyx
**Project:** https://github.com/Litlyx/litlyx
**Source:** https://github.com/Litlyx/litlyx
**Status:** done
**Compose source:** docker-compose.yml from repo, replaced build: with Docker Hub images

## What was done
- Created compose.yaml based on official docker-compose.yml
- Replaced build: directives with pre-built Docker Hub images (litlyx/litlyx-producer, litlyx/litlyx-consumer, litlyx/litlyx-dashboard)
- Includes MongoDB and Redis
- Created .env with credentials

## Issues
- None
