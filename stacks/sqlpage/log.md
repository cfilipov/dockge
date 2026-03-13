# SQLPage — Research Log

## Sources checked
1. `docker-compose.yml` in repo root — found but is a dev/test file (builds from source, test database profiles)
2. README.md — found `docker run` commands with image `lovasoa/sqlpage`

## Compose file origin
Converted from the `docker run` commands in the official GitHub README.

## Modifications
- Converted `docker run` flags to Compose V2 format
- Used the production-style volume layout (source + configuration directories)
- Added `restart: unless-stopped`
