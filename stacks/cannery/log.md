# Cannery

## Source
- Repository: https://codeberg.org/shibao/cannery
- Docker image: shibaobun/cannery

## Compose Source
- Fetched from: https://codeberg.org/shibao/cannery/raw/branch/stable/docker-compose.yml
- Official docker-compose.yml from the stable branch

## Changes from Original
- Removed `version: '3'` field (Compose V2)
- Removed commented-out nginx proxy manager and nginx-db services
- Fixed POSTGRES_USER/PASSWORD/DB env vars (removed erroneous quotes in values)
- Kept only the core cannery + postgres services

## Services
- **cannery**: Elixir-based ammunition tracking app (port 4000 internal)
- **cannery-db**: PostgreSQL 17 database
