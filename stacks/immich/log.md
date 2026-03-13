# Immich

- **Source**: https://github.com/immich-app/immich
- **Docker image**: ghcr.io/immich-app/immich-server, ghcr.io/immich-app/immich-machine-learning
- **Compose ref**: https://github.com/immich-app/immich/blob/main/docker/docker-compose.yml
- **Description**: High-performance self-hosted photo/video backup with ML-powered search, face recognition, and mobile apps
- **Services**: immich-server, immich-machine-learning, redis (Valkey), database (PostgreSQL with pgvectors)
- **Notes**: Removed pinned SHA digests for cleaner fixture. ML service handles face detection, CLIP embeddings, etc. Hardware acceleration available via hwaccel extensions.
