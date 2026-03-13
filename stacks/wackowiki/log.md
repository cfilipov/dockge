# WackoWiki Stack

## Source
- GitHub: WackoWiki/wackowiki
- Docker Hub: trojer/wackowiki

## Description
WackoWiki is a light and easy-to-install wiki engine written in PHP. Supports MySQL/MariaDB and PostgreSQL backends.

## Services
- **wackowiki** — PHP/Apache wiki application
- **db** — MariaDB database

## Volumes
- `wackowiki-data` — wiki application files
- `wackowiki-db` — database storage

## Ports
- 8080 → 80 (HTTP)
