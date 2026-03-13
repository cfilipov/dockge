# DAViCal

## Sources
- Official repo (GitLab): https://gitlab.com/davical-project/davical (no Docker files)
- Community Docker image: https://github.com/fintechstudios/davical-docker
- Docker Compose: https://github.com/fintechstudios/davical-docker/blob/master/docker-compose.yml
- Docker Hub: https://hub.docker.com/r/fintechstudios/davical

## Notes
- Official DAViCal project does not provide Docker images
- Using fintechstudios/davical community image (standalone, based on php:apache)
- Compose taken from the fintechstudios/davical-docker repo
- Removed `version: "3"` and `build: .` for Compose V2 compatibility
- Added postgres-data volume for persistence
- Runs on port 4080, default admin password: "admin"
