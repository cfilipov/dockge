# Calibre Web Automated

## Source
- GitHub: https://github.com/crocodilestick/Calibre-Web-Automated
- Docker Hub: crocodilestick/calibre-web-automated

## Research
- Found official docker-compose.yml in repository
- Single service with book ingest directory (files removed after processing)
- Simplified from upstream: removed optional plugin volume and commented-out options

## Compose
- Image: crocodilestick/calibre-web-automated:latest
- Port: 8083 (Web UI)
- Volumes: config, book-ingest, calibre-library
