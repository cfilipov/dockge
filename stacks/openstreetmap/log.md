# OpenStreetMap Website

- **Source**: https://github.com/openstreetmap/openstreetmap-website
- **Description**: The Rails application powering openstreetmap.org
- **Compose reference**: Official `docker-compose.yml` (adapted to use pre-built image instead of build context)
- **Services**: web (Rails app), db (PostGIS)
- **Default port**: 3000
- **Notes**: Official repo uses build context; this uses GHCR image for pre-built deployment
