# txtdot

## Source
- Website: https://txt.dc09.ru
- GitHub: https://github.com/txtdot/txtdot
- Docker image: `ghcr.io/tempoworks/txtdot:latest`

## Research
- docker-compose.yml found in repo root
- Image hosted on GitHub Container Registry
- Port 8080 for web interface
- Original compose mounts .env file but it's optional

## Compose
- Adapted from official docker-compose.yml in repo
- Removed `version: '3'` for Compose V2 compatibility
- Removed .env volume mount (not required for basic operation)
