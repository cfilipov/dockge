# blocky — Research Log

## Sources checked
1. `docker-compose.yml` / `docker-compose.yaml` in repo root — not found (404)
2. README.md — references Docker image `spx01/blocky`, links to installation docs
3. Installation docs (`https://0xerr0r.github.io/blocky/latest/installation/`) — found Docker Compose example

## Compose file origin
Taken from the official installation documentation (basic Docker Compose example).

## Modifications
- Removed `version: "2.1"` field (Compose V2 format)
- Removed `hostname` directive (not essential)
- Included a minimal `config.yml` based on the configuration docs, since the compose file bind-mounts it
