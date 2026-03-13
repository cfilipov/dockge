# Tube Archivist Stack

## Source
- GitHub: https://github.com/tubearchivist/tubearchivist
- Category: Media Streaming - Video Streaming

## Description
Tube Archivist is a self-hosted YouTube media server. It manages downloading, indexing, and searching your YouTube collections with a clean web interface.

## Stack Components
- **tubearchivist**: Main web application (Django/Python)
- **archivist-redis**: Redis for task queue and caching
- **archivist-es**: Elasticsearch for full-text search and indexing

## Notes
- Based on official docker-compose.yml from the repository
- Elasticsearch requires `vm.max_map_count=262144` on the host (sysctl)
- Media stored in /youtube volume, cache in /cache volume
- TA_HOST must match the URL used to access the web UI
- Elasticsearch password must match between tubearchivist and archivist-es services
- Custom ES image (bbilly1/tubearchivist-es) for amd64; use official ES 8.x for arm64
