# Agenta

## Sources
- Repository: https://github.com/agenta-ai/agenta
- Compose file: https://github.com/agenta-ai/agenta/blob/main/hosting/docker-compose/oss/docker-compose.gh.yml

## Notes
- Based on the official OSS Docker Compose file from `hosting/docker-compose/oss/docker-compose.gh.yml`
- Simplified from the full setup which includes multiple worker processes (evaluations, tracing, webhooks, events), cron, and Alembic migration runner
- Kept core services: web (frontend), api (backend), services, postgres, two redis instances, and supertokens (auth)
- Removed env_file references and traefik/nginx proxy profiles for standalone use
- Removed NewRelic monitoring wrapper from commands
- Original uses variable substitution for image names/tags; simplified to direct image references
