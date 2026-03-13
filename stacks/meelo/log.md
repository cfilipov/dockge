# Meelo

## Overview
Meelo is a self-hosted music server with metadata management, library scanning, and scrobbling support.

## Images
- `ghcr.io/arthi-chaud/meelo-server:latest` — API server
- `ghcr.io/arthi-chaud/meelo-front:latest` — web frontend
- `ghcr.io/arthi-chaud/meelo-scanner:latest` — library scanner
- `postgres:16-alpine` — database
- `getmeili/meilisearch:latest` — search engine

## Ports
- 5000 → 3000 (web UI via front)

## Notes
- Based on upstream docker-compose.yml, replaced build: with pre-built images
- Requires settings.json in CONFIG_DIR

## Source
- https://github.com/Arthi-chaud/Meelo
