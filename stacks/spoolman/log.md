# Spoolman

## Source
- Repository: https://github.com/Donkie/Spoolman
- Docker image: ghcr.io/donkie/spoolman:latest (also: donkieyo/spoolman:latest on Docker Hub)

## Compose Source
- From official wiki: https://github.com/Donkie/Spoolman/wiki/Installation
- Official docker-compose.yml example from installation guide

## Changes from Original
- Removed `version: '3.8'` field (Compose V2)
- Added container_name
- Changed TZ from Europe/Stockholm to UTC

## Notes
- Requires creating a `data` directory with ownership 1000:1000
- SQLite-based, no external database needed
- 3D printer filament spool manager

## Services
- **spoolman**: Filament spool management web app (port 7912 -> 8000)
