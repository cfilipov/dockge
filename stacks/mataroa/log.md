# Mataroa
**Project:** https://mataroa.blog
**Source:** https://github.com/mataroa-blog/mataroa
**Status:** failed
**Compose source:** https://github.com/mataroa-blog/mataroa/blob/main/docker-compose.yml

## What was done
- Found docker-compose.yml in repo — uses `build: .` (no pre-built image)
- Checked Docker Hub and ghcr.io — no official pre-built images
- Cannot create valid compose without a real registry image

## Issues
- Only build-from-source Docker support, no published container image
