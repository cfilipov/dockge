# OTOBO

## Source
- GitHub: https://github.com/RotherOSS/otobo
- Docker repo: https://github.com/RotherOSS/otobo-docker
- Compose: https://github.com/RotherOSS/otobo-docker/blob/rel-11_0/docker-compose/otobo-base.yml

## Research
- Official docker-compose from otobo-docker repo (multi-file setup)
- Merged base + HTTP override into single compose file
- Images: rotheross/otobo:latest-11_0, rotheross/otobo-elasticsearch:latest-11_0
- Five services: MariaDB + OTOBO web + OTOBO daemon + Elasticsearch + Redis
- Resolved env var substitutions to concrete defaults
- Web UI on port 80 (mapped to container port 5000)
- DB root password: otobo_root (change before production)
