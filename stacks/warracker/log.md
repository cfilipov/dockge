# Warracker

- **Status**: created
- **Source**: https://github.com/sassanix/Warracker
- **Image**: ghcr.io/sassanix/warracker/main:latest
- **Notes**: Warranty tracking application. Compose based on the official Docker/docker-compose.yml from the repo. Uses PostgreSQL backend. The official compose had build directives; replaced with the ghcr.io published image. Removed bind-mount for init.sql and migrations (handled internally by the image).
