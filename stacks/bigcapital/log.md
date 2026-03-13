# Bigcapital

- **Source**: https://github.com/bigcapitalhq/bigcapital
- **Images**: `bigcapitalhq/webapp:latest`, `bigcapitalhq/server:latest`, `mariadb:10`, `redis:7-alpine`, `envoyproxy/envoy:v1.30-latest`, `gotenberg/gotenberg:7`
- **Status**: created
- **Notes**: Based on the official `docker-compose.prod.yml`. Replaced `build:` directives for mysql and redis with standard images. Envoy config bind-mounted for API/webapp routing. The database_migration service from the original was omitted as it uses a `build:` directive with no published image.
