# Known
**Project:** https://withknown.com
**Source:** https://github.com/idno/known
**Status:** done
**Compose source:** https://github.com/idno/known/blob/dev/docker/docker-compose.yml

## What was done
- Found docker-compose.yml in repo's docker/ directory
- Created simplified compose.yaml with Known (jimwins/idno) + MySQL 8.0
- Removed Tailscale and Caddy services (not needed for testing)
- Created .env with database credentials

## Issues
- Original compose uses Tailscale for networking and Caddy as reverse proxy; simplified for testing
