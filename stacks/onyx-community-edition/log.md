# Onyx Community Edition

- **Status**: ok
- **Source**: https://github.com/onyx-dot-app/onyx
- **Images**: onyxdotapp/onyx-backend, onyxdotapp/onyx-web-server, onyxdotapp/onyx-model-server
- **Notes**: AI-powered document search and chat platform. Complex stack with 9 services: API server, background worker, web server, 2 model servers (inference + indexing), PostgreSQL, Vespa (search index), Redis (cache), MinIO (object storage), and nginx (reverse proxy). Requires nginx config files in ./nginx/ directory from the upstream repo (deployment/data/nginx/). The Vespa search engine needs significant memory. OpenSearch is omitted for simplicity but can be re-added.
