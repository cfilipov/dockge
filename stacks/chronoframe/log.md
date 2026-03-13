# ChronoFrame

- **Source**: https://github.com/HoshinoSuzumi/chronoframe
- **Docker image**: ghcr.io/hoshinosuzumi/chronoframe:latest
- **Compose ref**: https://github.com/HoshinoSuzumi/chronoframe/blob/main/docker-compose.yml
- **Description**: Self-hosted personal gallery with Live/Motion Photos support, EXIF parsing, geolocation recognition, and explore map
- **Services**: chronoframe (Nuxt.js app with embedded SQLite)
- **Notes**: Uses local storage provider by default (S3 also supported). Postgres optional but commented out in upstream.
