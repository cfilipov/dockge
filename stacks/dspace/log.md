# DSpace Stack

## Source
- Repository: https://github.com/DSpace/DSpace
- Compose file: https://github.com/DSpace/DSpace/blob/main/docker-compose.yml

## Research
- Found docker-compose.yml in the repo root
- Original has `build:` sections referencing Dockerfile.test and local Solr configs; removed these since pre-built images are available on Docker Hub
- Removed config bind-mount (`./dspace/config`) that requires source checkout
- Removed debug port (8000) and matomo integration (optional)
- Images: `dspace/dspace` (backend), `dspace/dspace-solr` (Solr with DSpace configsets), `postgres:15`

## Services
- **dspace**: DSpace backend REST API (port 8080)
- **dspacedb**: PostgreSQL 15 database
- **dspacesolr**: Solr search engine with DSpace cores (authority, oai, search, statistics, qaevent, suggestion, audit)

## Notes
- DSpace 7+ uses a separate Angular frontend (DSpace-angular) which is not included here
- The backend waits for PostgreSQL to be ready before starting
- Uses custom network with specific subnet for trusted proxy configuration
