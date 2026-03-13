# 015

- **Source**: https://github.com/keven1024/015
- **Category**: File Transfer - Single-click & Drag-n-drop Upload
- **Description**: Open-source temporary file sharing platform supporting upload, download, and sharing of files and text.
- **Image(s)**: `fudaoyuanicu/015-app:latest`, `fudaoyuanicu/015-worker:latest`, `redis:7`
- **Compose source**: Adapted from upstream `docker-compose.yml`
- **Config files**: `config.yaml` (app configuration with Redis, upload, and site settings)
- **Notes**: Requires Redis for queue processing. Worker handles background tasks. Config file is bind-mounted read-only.
