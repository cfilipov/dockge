# Alf.io

## Sources
- GitHub: https://github.com/alfio-event/alf.io
- Docker Compose: https://raw.githubusercontent.com/alfio-event/alf.io/master/docker-compose.yml
- Docker Hub: https://hub.docker.com/r/alfio/alf.io

## Notes
- Compose file taken from official repo (master branch)
- Removed `version: "3.7"` field for Compose V2 compatibility
- Added `depends_on` with healthcheck condition so alfio waits for postgres
- Added healthcheck to postgres service
- Simplified port syntax for postgres from long form to short form
- Image `alfio/alf.io` is the official Docker Hub image
- PostgreSQL 10 is what the upstream repo specifies
