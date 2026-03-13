# Tiki Stack

## Source
- Docker Hub: tikiwiki/tikiwiki
- Project: https://tiki.org/

## Description
Tiki Wiki CMS Groupware is a full-featured web application with wiki, forums, blogs, file galleries, trackers, and more. Requires MySQL/MariaDB.

## Services
- **tiki** — PHP/Apache application server
- **db** — MariaDB database

## Volumes
- `tiki-data` — application files
- `tiki-db` — database storage

## Ports
- 8080 → 80 (HTTP)
