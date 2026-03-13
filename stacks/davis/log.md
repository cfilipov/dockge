# Davis

## Sources
- GitHub repo: https://github.com/tchapi/davis
- Docker Compose files: https://github.com/tchapi/davis/tree/main/docker/
- Standalone compose: docker/docker-compose-standalone.yml
- Environment template: docker/.env

## Notes
- Davis provides two Docker image variants: standalone (with Caddy) and barebone (FPM only)
- Using the standalone variant (ghcr.io/tchapi/davis-standalone) for simplicity
- Compose adapted from docker/docker-compose-standalone.yml in the repo
- Removed `version`, `build`, `container_name`, and `name` fields for Compose V2
- .env file adapted from the repo's docker/.env template with simplified defaults
- Runs on port 9000, admin credentials: admin/admin
