# Swetrix

- **Source**: https://github.com/swetrix/swetrix + https://github.com/swetrix/selfhosting
- **Status**: ok
- **Images**: swetrix/swetrix-fe:v5.0.3, swetrix/swetrix-api:v5.0.3, redis:8.2-alpine, clickhouse/clickhouse-server:24.10-alpine, nginx:1.29.4-alpine
- **Notes**: Based on the official selfhosting repo (swetrix/selfhosting, main branch, compose.yaml). All ClickHouse XML configs and nginx proxy config reproduced from upstream. 5-service stack: frontend, API, Redis, ClickHouse, nginx reverse proxy.
