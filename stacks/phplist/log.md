# phpList Stack

## Sources
- Repository: https://github.com/phpList/phplist3
- Docker image: phplist/phplist (Docker Hub)
- Dockerfile: https://github.com/phpList/phplist3/blob/master/Dockerfile.release

## Notes
phpList is an open-source newsletter and email marketing manager. The Docker image
(`phplist/phplist`) is built from `Dockerfile.release` in the repo using Debian Bookworm
with Apache and PHP. It exposes port 80. No official docker-compose file exists in the
repository, so this compose file was constructed based on the Docker image requirements
(Apache on port 80, needs MySQL/MariaDB database). Environment variables for database
connection follow phpList's standard configuration pattern.

## Services
- **phplist** (`phplist/phplist:latest`) - phpList newsletter manager (port 8080->80)
- **db** (`mariadb:11`) - MariaDB database
