# Halo — Research Log

## Sources checked
1. GitHub repo `halo-dev/halo` — no compose file in repo root
2. README.md — found `docker run` command with image `halohub/halo:2.22`

## Compose file origin
Converted from the `docker run` command in the official GitHub README.

## Modifications
- Converted `docker run` flags to Compose V2 format
- Changed host volume path (`~/.halo2`) to relative path (`./data`)
- Added `restart: unless-stopped`
