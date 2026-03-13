# LiveCodes

## Status: SKIPPED

## Sources
- GitHub: https://github.com/live-codes/livecodes
- docker-compose.yml: https://github.com/live-codes/livecodes/blob/develop/docker-compose.yml
- Self-hosting docs: https://livecodes.io/docs/features/self-hosting

## Research Notes
- docker-compose.yml exists but the main app service uses `build: .` (builds from source)
- Docker Hub image `livecodes/livecodes` exists but has 0 pulls
- Requires Caddy config files from `./server/caddy/` directory
- No pre-built Docker image available for simple deployment
- Skipped: requires building from source with multiple config file dependencies
