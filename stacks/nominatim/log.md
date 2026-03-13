# Nominatim

- **Source**: https://github.com/osm-search/Nominatim
- **Docker image**: mediagis/nominatim (https://github.com/mediagis/nominatim-docker)
- **Description**: Open-source geocoding with OpenStreetMap data (search by name, reverse geocode)
- **Compose reference**: Constructed from Docker Hub documentation and example.md
- **Services**: nominatim (includes embedded PostgreSQL + PostGIS)
- **Default port**: 8080
- **Notes**: PBF_URL set to Monaco for minimal test data; increase memory settings for larger regions
