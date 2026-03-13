# Ente

- **Source**: https://github.com/ente-io/ente
- **Docker image**: ghcr.io/ente-io/server:latest
- **Compose ref**: https://github.com/ente-io/ente/blob/main/server/compose.yaml
- **Description**: End-to-end encrypted photo storage and sharing platform (Google Photos alternative)
- **Services**: museum (API server), socat (port relay), postgres, minio (S3-compatible storage)
- **Notes**: Replaced build: with prebuilt image. Socat bridges localhost:3200 inside museum container to minio. Config files (museum.yaml, credentials.yaml) are bind-mounted.
