# motionEye

## Sources
- Repo docker-compose.yml: https://github.com/motioneye-project/motioneye/blob/dev/docker/docker-compose.yml
- Docker README: https://github.com/motioneye-project/motioneye/blob/dev/docker/README.md
- Docker image: ghcr.io/motioneye-project/motioneye:edge

## Notes
- Compose from official repo docker/ directory
- Removed `version: "3.5"` field (Compose V2)
- Added restart policy and container_name
- Ports: 8765 (web UI), 8081 (streaming)
- Named volumes for config (/etc/motioneye) and recordings (/var/lib/motioneye)
- Image tag is "edge" (dev builds); stable releases not yet available per repo comments
