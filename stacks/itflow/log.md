# ITFlow

## Source
- GitHub: https://github.com/itflow-org/itflow
- Docker repo: https://github.com/itflow-org/itflow-docker
- Docker Hub: https://hub.docker.com/r/itfloworg/itflow
- Docs: https://docs.itflow.org/installation_docker

## Research
- Official docker-compose.yml from itflow-docker repo
- Image: itfloworg/itflow:latest
- Two services: MariaDB 10.11.6 + ITFlow app
- Simplified: removed external networks, container_name, hostname directives
- Replaced env var substitution with defaults for standalone use
- Web UI on port 8080
- Docker support is community-maintained (not officially supported)
