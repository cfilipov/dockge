# Onyx Community Edition

## Sources
- Repository: https://github.com/onyx-dot-app/onyx
- Compose file: https://github.com/onyx-dot-app/onyx/blob/main/deployment/docker_compose/docker-compose.yml

## Notes
- Based on the official docker-compose.yml from deployment/docker_compose/
- Simplified from the full setup which includes OpenSearch, MinIO, and code-interpreter
- Core services: API server, background worker, web server, inference model server, indexing model server, PostgreSQL, Vespa (search), Redis, and nginx
- Removed OpenSearch (optional), MinIO (optional S3 file store), and code-interpreter
- Removed nginx config volume mount (requires config file from repo's deployment/data/nginx/)
- Access at http://localhost:3000
- The full deployment requires significant resources (Vespa, model servers, etc.)
