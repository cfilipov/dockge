# InvenioRDM Stack

## Source
- Repository: https://github.com/inveniosoftware/demo-inveniordm
- Docker services: https://github.com/inveniosoftware/demo-inveniordm/blob/master/docker-services.yml
- Documentation: https://inveniordm.docs.cern.ch

## Research
- Main repo (invenio-app-rdm) has no pre-built Docker image on Docker Hub
- Demo repo (demo-inveniordm) has docker-compose files using `extends` from docker-services.yml
- The app service requires building from source with `invenio-cli`
- Infrastructure services (Redis, PostgreSQL, RabbitMQ, OpenSearch) use standard images
- Inlined the service definitions from docker-services.yml, removing `extends` and CERN registry prefixes

## Services
- **cache**: Redis 7 (port 6379)
- **db**: PostgreSQL 12.4 (port 5432)
- **mq**: RabbitMQ with management UI (ports 5672, 15672)
- **search**: OpenSearch 2.15.0 (ports 9200, 9600)
- **opensearch-dashboards**: OpenSearch Dashboards (port 5601)

## Notes
- This is the infrastructure-only stack; InvenioRDM application itself must be built from source using `invenio-cli`
- Based on the official docker-services.yml from the demo-inveniordm repository
- The app connects to these services via environment variables (INVENIO_SQLALCHEMY_DATABASE_URI, INVENIO_BROKER_URL, etc.)
