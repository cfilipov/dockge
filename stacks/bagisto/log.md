# Bagisto Stack

## Research
- Found docker-compose.yml in bagisto/bagisto GitHub repo (Laravel Sail based)
- Services: app, MySQL 8.0, Redis, Elasticsearch 7.17, Kibana, Mailpit
- Original uses `build:` for app; replaced with `image: bagisto/bagisto:latest`

## Changes from upstream
- Replaced `build:` with `image: bagisto/bagisto:latest`
- Removed bind-mount volumes referencing local source
- Added named volume for app data
- Kept all supporting services with real Docker Hub images
