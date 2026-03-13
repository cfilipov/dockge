# osem

## Sources
- Repository: https://github.com/openSUSE/osem
- Compose file: https://github.com/openSUSE/osem/blob/master/docker-compose.yml
- Docker image: registry.opensuse.org/opensuse/infrastructure/dale/containers/osem/base:latest

## Notes
- Original compose uses `build: .` which builds from local Dockerfile
- The Dockerfile extends `registry.opensuse.org/opensuse/infrastructure/dale/containers/osem/base:latest`
- No pre-built production image on Docker Hub or GHCR; using the openSUSE registry base image
- Removed source code volume mount (`.:/osem`) since this is not a dev setup
- Added persistent volume for PostgreSQL data
- App accessible on port 3000
- This is a Ruby on Rails application (conference management for openSUSE events)
