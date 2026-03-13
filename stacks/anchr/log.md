# Anchr

- **Status**: ok
- **Source**: https://github.com/muety/anchr
- **Image**: ghcr.io/muety/anchr (GitHub Container Registry)
- **Compose source**: https://github.com/muety/anchr/blob/master/docker-compose.yml (adapted - replaced build: directive with image)
- **Notes**: Link shortener and bookmark manager. Original compose uses `build: ./` which was replaced with ghcr.io image. Removed mongo-init.sh bind mount since it's repo-specific.
