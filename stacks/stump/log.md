# Stump

## Source
- GitHub: https://github.com/stumpapp/stump
- Docker Hub: aaronleopold/stump

## Research
- Found Docker setup info at stumpapp.dev/installation/docker
- Comics, manga, and digital book server
- Simple single-service setup with config and data volumes

## Compose
- Image: aaronleopold/stump:latest
- Port: 10801 (Web UI)
- Volumes: config, data (media library)
- Environment: PUID, PGID, STUMP_CONFIG_DIR
