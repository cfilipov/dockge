# Modoboa

- **Status**: partial
- **Source**: https://github.com/modoboa/modoboa
- **Images**: redis:8-alpine, postgres:16-alpine, dovecot/dovecot:2.3-latest
- **Notes**: Modoboa's official docker-compose.yml uses `build:` directives for the main app (api, front, rq, radicale, policyd, amavis) with no published registry images. Only the supporting services (Redis, PostgreSQL, Dovecot) use real images. The core Modoboa application has no official Docker Hub image, so this compose only includes the infrastructure services. Full deployment requires building from source.
