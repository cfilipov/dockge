# Quassel IRC

## Sources
- LinuxServer.io Docker image: https://github.com/linuxserver/docker-quassel-core
- Image: lscr.io/linuxserver/quassel-core:latest
- Compose example from LinuxServer README

## Notes
- LinuxServer image is archived/deprecated but still available
- No official Docker image from the Quassel project
- Port 4242 for Quassel client connections, 113 for Ident
- Config/database volume at /config
- Supports SQLite (default) or PostgreSQL backend
