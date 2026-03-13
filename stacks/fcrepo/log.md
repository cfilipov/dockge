# Fedora Commons Repository Stack

## Source
- Repository: https://github.com/fcrepo/fcrepo
- Docker image repo: https://github.com/fcrepo-exts/fcrepo-docker
- Docker Hub: https://hub.docker.com/r/fcrepo/fcrepo
- Quick Start: https://wiki.lyrasis.org/display/FEDORA6x/Quick+Start

## Research
- No docker-compose file in the main repo
- Official Docker image exists: `fcrepo/fcrepo`
- Quick Start docs provide `docker run` command: `docker run -p8080:8080 --name=fcrepo fcrepo/fcrepo`
- fcrepo-docker README documents environment variables and volume paths
- Converted docker run command to compose format

## Services
- **fcrepo**: Fedora Commons Repository on Tomcat (port 8080)

## Access
- URL: http://localhost:8080/fcrepo
- Default credentials: fedoraAdmin / fedoraAdmin

## Notes
- Data stored at `/usr/local/tomcat/fcrepo-home` inside the container
- Uses embedded database by default; can be configured with external PostgreSQL/MySQL via CATALINA_OPTS
