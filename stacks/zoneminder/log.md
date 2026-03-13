# Zoneminder

## Sources
- Docker repo: https://github.com/ZoneMinder/zmdockerfiles
- docker-compose.yaml from repo master branch
- Docker image: zoneminderhq/zoneminder:latest-ubuntu18.04

## Notes
- Compose from official zmdockerfiles repository
- Removed `version: '3.1'` field (Compose V2)
- Changed default TZ from Australia/Perth to America/New_York
- Port 7878 maps to internal port 80 (web UI)
- shm_size 512M for shared memory
- Privileged mode enabled for hardware access
- Four named volumes: events, images, mysql, logs
- Hardware acceleration available via /dev/dri device (not included by default)
