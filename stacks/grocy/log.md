# grocy Stack Research Log

## Sources
- Main repo: https://github.com/grocy/grocy
- Docker image repo: https://github.com/linuxserver/docker-grocy
- Docker Hub: https://hub.docker.com/r/linuxserver/grocy
- Reference compose from: https://raw.githubusercontent.com/linuxserver/docker-grocy/master/README.md

## Notes
- Image: `lscr.io/linuxserver/grocy:latest` (LinuxServer.io maintained)
- Self-contained single container (PHP + SQLite, no external DB needed)
- Default credentials: admin / admin
- Port: 9283 -> 80
- Config persisted in /config volume
- Alpine-based with PHP support
