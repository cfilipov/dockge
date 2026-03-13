# LibrePhotos

- **Source**: https://github.com/LibrePhotos/librephotos
- **Docker image**: reallibrephotos/librephotos, reallibrephotos/librephotos-frontend, reallibrephotos/librephotos-proxy
- **Compose ref**: https://github.com/LibrePhotos/librephotos-docker/blob/main/docker-compose.yml
- **Description**: Self-hosted Google Photos alternative with face recognition, object detection, and timeline grouping
- **Services**: proxy (nginx), frontend (React), backend (Django), db (PostgreSQL with auto-upgrade)
- **Notes**: Four-service architecture. Backend handles ML processing. Scan directory is bind-mounted for photo import.
