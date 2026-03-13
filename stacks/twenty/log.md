# Twenty

## Sources
- GitHub repo: https://github.com/twentyhq/twenty
- Compose file: https://github.com/twentyhq/twenty/blob/main/packages/twenty-docker/docker-compose.yml
- Docker image: twentycrm/twenty

## Notes
- Compose taken verbatim from the official Twenty repository (packages/twenty-docker/docker-compose.yml)
- Includes 4 services: server, worker, PostgreSQL 16, and Redis
- Worker runs background jobs; migrations and cron run on the server only
- APP_SECRET should be changed from the default placeholder before production use
- Access at http://localhost:3000
