# Inventaire

## Source
- Repository: https://codeberg.org/inventaire/inventaire
- Docker repo: https://codeberg.org/inventaire/docker-inventaire
- Docker image: inventaire/inventaire:latest

## Compose Source
- Fetched from: https://codeberg.org/inventaire/docker-inventaire/raw/branch/main/docker-compose.yml
- Official docker-compose.yml from the docker-inventaire repository

## Changes from Original
- Removed `version` field (Compose V2)
- Replaced custom-built couchdb (Dockerfile.couchdb) with standard `couchdb:3` image
- Removed nginx and certbot services (reverse proxy not part of core app)
- Removed env_file references (not including .env template)
- Added COUCHDB_USER/PASSWORD env vars for couchdb initialization
- Used named volumes instead of bind mounts

## Services
- **inventaire**: Book/resource inventory web app (Node.js)
- **couchdb**: CouchDB 3 database
- **elasticsearch**: Elasticsearch 7.17 for search functionality
