# BookLore

## Source
- GitHub: https://github.com/booklore-app/booklore
- Docker Hub: booklore/booklore

## Research
- README contains detailed Docker compose information
- Two services: booklore app + MariaDB 11.4.5
- Supports LOCAL and NETWORK disk types
- MariaDB healthcheck included

## Compose
- Images: booklore/booklore:latest, mariadb:11.4.5
- Port: 6060 (Web UI)
- .env file for database credentials
- Named volume for MariaDB data persistence
