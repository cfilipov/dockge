# Wiki.js Stack

## Source
- GitHub: requarks/wiki
- Docker Hub: linuxserver/wikijs (LinuxServer.io wrap of requarks/wiki)

## Description
Wiki.js is a powerful and extensible open-source wiki software built on Node.js. Features a modern WYSIWYG editor, Git-backed storage, and extensive authentication options. Uses PostgreSQL for data storage.

## Services
- **wikijs** — Wiki.js application (Node.js)
- **db** — PostgreSQL database

## Volumes
- `wikijs-config` — application configuration
- `wikijs-data` — wiki content and assets
- `wikijs-db` — PostgreSQL data

## Ports
- 3000 → 3000 (HTTP)
