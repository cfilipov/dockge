# Komga

## Source
- GitHub: https://github.com/gotson/komga
- Docker Hub: gotson/komga

## Research
- Found official compose example at komga.org/docs/installation/docker
- Media server for comics, mangas, BD, magazines, eBooks
- Uses user directive instead of PUID/PGID

## Compose
- Image: gotson/komga:latest
- Port: 25600 (Web UI and API)
- Volumes: config (database, must be local filesystem), data (media library)
