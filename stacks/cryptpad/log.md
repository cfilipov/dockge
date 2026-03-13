# CryptPad

## Sources
- Official docker-compose.yml from repo: https://github.com/cryptpad/cryptpad/blob/main/docker-compose.yml
- Docker Hub image: `cryptpad/cryptpad`

## Notes
- Compose file taken directly from the official repository
- Ports 3000 (main) and 3003 (websocket) exposed
- OnlyOffice integration available but requires separate license acceptance (omitted)
- Original uses bind mounts; converted to named volumes for portability
- ulimits set to 1M file descriptors as recommended
