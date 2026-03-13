# Chyrp Lite
**Project:** https://chyrp-lite.net
**Source:** https://github.com/xenocrat/chyrp-lite
**Status:** failed
**Compose source:** https://github.com/xenocrat/chyrp-lite/blob/master/docker-compose.yaml

## What was done
- Found docker-compose.yaml in repo — uses `build: .` (no pre-built image)
- Checked Docker Hub and ghcr.io — no pre-built images exist
- Cannot create valid compose without a real registry image

## Issues
- Only build-from-source Docker support, no published container image
