# OpenRouteService

- **Source**: https://github.com/GIScience/openrouteservice
- **Description**: Open-source route planner with directions, isochrones, and matrix calculations
- **Compose reference**: Official `docker-compose.yml` (simplified, removed build context)
- **Services**: ors-app (Java routing engine)
- **Default ports**: 8080 (API), 9001 (monitoring)
- **Notes**: Ships with example Heidelberg PBF; place custom PBF files in /home/ors/files/ volume
