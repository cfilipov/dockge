# Vendure Stack

## Research
- GitHub: vendure-ecommerce/vendure - TypeScript/Node.js headless commerce
- Found docker-compose.yml in master branch: multi-DB dev setup (MariaDB, MySQL 5/8, Postgres 12/16, Keycloak, Elasticsearch, Redis)
- No official pre-built Docker image
- Simplified to just Vendure app + PostgreSQL 16

## Compose
- Used `node:20-alpine` (no official Vendure image)
- PostgreSQL 16 for database (Vendure supports multiple DBs, Postgres recommended)
- Minimal configuration with superadmin credentials
