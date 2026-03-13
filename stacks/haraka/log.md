# Haraka

## Sources
- GitHub repo: https://github.com/haraka/Haraka
- Docker image repo: https://github.com/instrumentisto/haraka-docker-image
- Docker Hub: https://hub.docker.com/r/instrumentisto/haraka (17k+ pulls)

## Image
- `instrumentisto/haraka:latest` (well-maintained community image, multi-arch)

## Notes
- The Haraka repo itself has a Dockerfile but it's user-contributed and outdated (phusion/baseimage)
- Used instrumentisto/haraka which is actively maintained with proper docs
- Compose derived from docker run example in Docker Hub README
- Configuration via mounted files in /etc/haraka/config/
- Plugins installable via HARAKA_INSTALL_PLUGINS env var (comma-separated NPM packages)
- Exposes SMTP on port 25
