# Shlink

## Sources
- GitHub repo: https://github.com/shlinkio/shlink
- Docker Hub image: shlinkio/shlink
- Official docs: https://shlink.io/documentation/install-docker-image/

## Notes
- Compose based on official Docker documentation's docker run example
- Uses built-in SQLite by default (supports MySQL, PostgreSQL, MariaDB, MSSQL)
- Port 8080 exposed
- Optional GEOLITE_LICENSE_KEY for IP geolocation
- Repo's docker-compose.yml is a dev setup (builds from source with all DB engines); not used
