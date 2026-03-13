# Docspell

## What was done
- Based on official docker-compose from github.com/docspell/docker
- Services: restserver (web UI + API), joex (job executor), consumedir (file watcher), PostgreSQL, Solr (full-text search)
- All credentials extracted to .env file
- Images: ghcr.io/docspell/restserver, ghcr.io/docspell/joex, docspell/dsc, postgres:16, solr:9
- Ports: 7880 (web UI), 7878 (joex)
