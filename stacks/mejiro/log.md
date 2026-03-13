# Mejiro (Pellicola)

- **Source**: https://github.com/dmpop/pellicola
- **Docker image**: idocker1688/pellicola:latest (community image)
- **Compose ref**: https://github.com/dmpop/pellicola/blob/main/docker-compose.yml
- **Description**: Minimalist PHP photo gallery with thumbnails, EXIF display, pagination, and optional map view
- **Services**: pellicola (PHP/Apache)
- **Notes**: Upstream compose uses build: and Caddy reverse proxy — simplified to prebuilt image with direct port exposure. Config via PHP file. Formerly known as Mejiro, renamed to Pellicola.
