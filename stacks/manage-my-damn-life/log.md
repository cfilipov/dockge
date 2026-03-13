# Manage My Damn Life

## Sources
- GitHub repo: https://github.com/intri-in/manage-my-damn-life-nextjs
- Docker Compose sample: docker-compose.yml.sample in repo root
- Environment template: sample.env.local in repo root
- Docker Hub: https://hub.docker.com/r/intriin/mmdl

## Notes
- Compose adapted from docker-compose.yml.sample in the repo
- Removed custom network (app-tier) — default compose network suffices
- Added db-data volume for MySQL persistence
- .env adapted from sample.env.local with MySQL dialect (matching the compose sample)
- Runs on port 3000
- DOCKER_INSTALL=true enables Docker-specific behavior
