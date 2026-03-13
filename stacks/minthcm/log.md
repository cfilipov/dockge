# MintHCM

## Sources
- Docker Compose from official repo: https://github.com/minthcm/minthcm/blob/master/docker/docker-compose.yml
- .env file from official repo: https://github.com/minthcm/minthcm/blob/master/docker/.env
- Docker image: minthcm/minthcm on Docker Hub

## Notes
- Three services: MintHCM web app, Percona Server 8 (MySQL), Elasticsearch 7.9.3
- Web service uses env_file for configuration (DB host, credentials, Elasticsearch settings)
- DB healthcheck ensures web service starts only after DB is ready
- Elasticsearch configured as single-node with 4GB memory limit
- Default credentials: admin / minthcm
- Renamed container service names from `minthcm-web` etc. to simpler `web`, `db`, `elasticsearch`
- Updated DB_HOST and ELASTICSEARCH_HOST in .env to match new service names
