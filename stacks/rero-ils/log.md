# RERO ILS - Research Log

## Sources
1. **Project homepage**: https://ils.test.rero.ch/ - Version 1.27.0, links to GitHub repo
2. **GitHub repo**: https://github.com/rero/rero-ils - Contains Dockerfile, docker-compose.yml, docker-compose.full.yml, docker-services.yml
3. **docker-services.yml**: https://github.com/rero/rero-ils/blob/staging/docker-services.yml - Base service definitions
4. **docker-compose.full.yml**: https://github.com/rero/rero-ils/blob/staging/docker-compose.full.yml - Production-like setup
5. **Docker Hub**: https://hub.docker.com/r/rero/rero-ils - 68k+ pulls, tags: ils-dev, bib-v1.27.1, ils-test

## Notes
- Built on the Invenio framework (CERN's digital repository framework)
- Architecture: UI app (uWSGI), REST API (uWSGI), Celery worker, Celery beat scheduler
- Infrastructure: PostgreSQL 17, Redis (cache/sessions), RabbitMQ (message broker), Elasticsearch 7.10.2, Kibana, Flower
- Original compose uses `extends:` from docker-services.yml and `build:` contexts; adapted to use published Docker Hub image
- The original repo also has HAProxy (lb) and Nginx (frontend) services; omitted as they require custom build contexts
- The Elasticsearch service in the original uses a custom build with ICU plugin; substituted with the standard ES OSS image
- RERO ILS companion images also on Docker Hub: rero-ils-nginx (6.7k pulls), rero-ils-es (1.8k pulls)
