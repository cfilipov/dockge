# farmOS Stack Research Log

## Sources
- GitHub repo: https://github.com/farmOS/farmOS (branch: 4.x)
- Reference compose: https://github.com/farmOS/farmOS/blob/4.x/docker/docker-compose.production.yml
- Docker docs: https://farmos.org/hosting/docker/
- Docker Hub: https://hub.docker.com/r/farmos/farmos

## Notes
- Image: `farmos/farmos` (pin to specific version, avoid `latest`)
- Database: PostgreSQL 17
- Data persisted via /opt/drupal/web/sites volume
- Keys directory for OAuth2 keypair files
- Docs recommend bind mounts; converted to named volumes for portability
- After first start, complete setup via web installer
