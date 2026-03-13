# Funkwhale

- **Source**: https://dev.funkwhale.audio/funkwhale/funkwhale
- **Docker image**: funkwhale/api, funkwhale/front
- **Reference**: https://dev.funkwhale.audio/funkwhale/funkwhale/-/blob/develop/deploy/docker-compose.yml
- **Description**: Federated music streaming platform (ActivityPub). Multi-service: API, Celery workers, Nginx frontend, PostgreSQL, Redis.
- **Ports**: 5000 (web UI via nginx)
- **Volumes**: PostgreSQL data, Redis data, music, media uploads, static files
