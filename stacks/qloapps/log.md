# QloApps

## Sources
- GitHub: https://github.com/Qloapps/QloApps
- Docker repo: https://github.com/webkul/qloapps_docker
- Docker Hub: https://hub.docker.com/r/webkul/qloapps_docker
- docker run command from Docker repo README

## Notes
- Converted `docker run` command to compose format
- Monolithic container (Apache + MySQL + PHP all in one)
- Image based on Ubuntu 18.04 with MySQL 5.7 and PHP 7.2
- Removed SSH port mapping (port 2222:22) as not needed for typical use
- Uses .env file for passwords and database name
- Official image from webkul (QloApps maintainer)
