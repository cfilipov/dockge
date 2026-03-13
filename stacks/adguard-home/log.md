# AdGuard Home — Research Log

## Sources checked
1. `docker-compose.yml` / `docker-compose.yaml` in repo root — not found (404)
2. README.md — references Docker Hub image `adguard/adguardhome` but no compose example
3. GitHub Wiki Docker page (`https://github.com/AdguardTeam/AdGuardHome/wiki/Docker`) — found full `docker run` command

## Compose file origin
Converted from the `docker run` command on the official GitHub wiki Docker page.

## Modifications
- Converted `docker run` flags to Compose V2 format
- Changed host volume paths (`/my/own/workdir`, `/my/own/confdir`) to relative paths (`./work`, `./conf`)
- Removed `version:` field (Compose V2 format)
