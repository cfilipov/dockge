# Bluecherry

## Sources
- Docker repo: https://github.com/bluecherrydvr/bluecherry-docker
- docker-compose.yml from repo master branch
- .env-org from repo master branch
- Docker Hub image: bluecherrydvr/bluecherry

## Notes
- Compose adapted from official bluecherry-docker repository
- Removed unrelated uptime-kuma and mta_mailer services (not core to Bluecherry)
- Changed image tag from dev-ci to latest for general use
- Removed `version: '3.7'` field (Compose V2)
- MySQL 8.0 with healthcheck, Bluecherry server depends on healthy DB
- Ports: 7001 (web UI), 7002 (RTSP)
- .env file based on .env-org from repo with DB credentials
