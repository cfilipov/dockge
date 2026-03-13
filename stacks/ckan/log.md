# CKAN
**Project:** https://github.com/ckan/ckan
**Source:** https://github.com/ckan/ckan-docker
**Status:** done
**Compose source:** ckan/ckan-docker repo docker-compose.yml (adapted to use pre-built images)
## What was done
- Created compose.yaml based on official ckan-docker repo
- Replaced build: directives with pre-built images (ckan/ckan-base, postgres:14-alpine)
- Includes ckan, datapusher, db, solr, redis services
- Created .env with all required variables from .env.example
## Issues
- Omitted nginx proxy to simplify
- ckan/ckan-base image may need custom Dockerfile in production; used base image for testing
