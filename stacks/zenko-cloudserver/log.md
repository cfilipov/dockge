# Zenko CloudServer

## Sources
- Repository: https://github.com/scality/cloudserver (branch: development/9.2)
- Dockerfile: https://github.com/scality/cloudserver/blob/development/9.2/Dockerfile
- Docker Hub: https://hub.docker.com/r/zenko/cloudserver
- README: https://github.com/scality/cloudserver

## Notes
- Image: `zenko/cloudserver:latest`
- S3-compatible object storage server
- Port 8000 is the main S3 API endpoint (from Dockerfile EXPOSE)
- Default credentials: accessKey1 / verySecretKey1 (from README)
- S3DATA=file for persistent file-based storage (default is memory)
- Volumes for data and metadata persistence (from Dockerfile VOLUME directives)
- No official docker-compose.yml in repo; composed from Dockerfile and README docs
