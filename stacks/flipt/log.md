# Flipt

## Source
- Repository: https://github.com/flipt-io/flipt
- Docker run command from README: `docker run --rm -p 8080:8080 -p 9000:9000 -t docker.flipt.io/flipt/flipt:latest`

## Notes
- The repo's docker-compose.yml is dev-only (uses `build` context), so compose was derived from the official docker run command in the README
- Single-binary application with embedded SQLite by default
- Image: `docker.flipt.io/flipt/flipt:latest` (custom registry)
- Port 8080: HTTP API + UI
- Port 9000: gRPC API
- Added a named volume for data persistence at `/var/opt/flipt`
