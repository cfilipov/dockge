# Pydio Cells

## Source
- GitHub: https://github.com/pydio/cells
- Docker Hub: https://hub.docker.com/r/pydio/cells

## Description
Pydio Cells is a future-proof self-hosted file sharing platform. It provides file management, sharing, and collaboration features with a modern web interface.

## Stack Components
- **cells**: Pydio Cells server (pydio/cells:4)
- **mariadb**: MariaDB database backend (mariadb:11)

## Ports
- 8080: Pydio Cells web interface

## Volumes
- cells_data: Application data
- cells_workdir: Working directory / file storage
- mariadb_data: Database storage

## Configuration Notes
- CELLS_SITE_EXTERNAL must be set to the public-facing URL
- Database credentials shared between cells and mariadb services via .env
- Latest stable v4 branch used (v5 is alpha)
