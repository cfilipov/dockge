# Plane

- **Status**: ok
- **Source**: https://github.com/makeplane/plane
- **Image**: makeplane/plane-frontend:latest, makeplane/plane-backend:latest, makeplane/plane-proxy:latest
- **Notes**: Open-source project management tool (Jira alternative). Upstream docker-compose.yml uses `build:` directives; replaced with official Docker Hub images (makeplane/plane-*). Multi-service architecture: frontend, space, admin, API, worker, beat-worker, proxy (nginx), PostgreSQL, Redis, MinIO. The proxy service handles routing between frontend services.
