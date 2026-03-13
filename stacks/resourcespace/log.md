# ResourceSpace

## Source
- GitHub: https://github.com/resourcespace/resourcespace
- Docker Hub: https://hub.docker.com/r/suntorytimed/resourcespace

## Description
Open source digital asset management system. Web-based platform for organizing, storing, and sharing digital assets.

## Stack
- suntorytimed/resourcespace:latest — main application (port 80)
- mariadb:11 — database

## Notes
- Community Docker image by suntorytimed (no official image)
- Requires MariaDB for database storage
- Configuration stored in /var/www/html/include volume
- Digital assets stored in /var/www/html/filestore volume
