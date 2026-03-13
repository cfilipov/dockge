# Part-DB

## Source
- Repository: https://github.com/Part-DB/Part-DB-server
- Docker image: jbtronics/part-db1:latest
- Documentation: https://docs.part-db.de/installation/installation_docker.html

## Compose Source
- From official documentation: https://docs.part-db.de/installation/installation_docker.html
- Used the MySQL variant (docs also show SQLite-only option)

## Changes from Original
- Removed `version: '3.3'` field (Compose V2)
- Added container_name for database service
- Used placeholder passwords instead of "SECRET_USER_PASSWORD"

## Services
- **partdb**: Part-DB electronic parts inventory manager (port 8080)
- **database**: MySQL 8.0 database backend
