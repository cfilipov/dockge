# FreeScout

## Source
- GitHub: https://github.com/freescout-helpdesk/freescout
- Docker image: https://github.com/tiredofit/docker-freescout
- Compose example: https://github.com/tiredofit/docker-freescout/blob/main/examples/compose.yml

## Research
- No official Docker image from freescout-helpdesk org
- Community image by tiredofit/freescout widely used
- Compose example from tiredofit repo's examples/compose.yml
- Simplified: removed external networks, backup service, container_name directives
- Two services: MariaDB + FreeScout app
- Default admin: admin@admin.com / freescout
- Web UI on port 8080
