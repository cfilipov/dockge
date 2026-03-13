# Svix

- **Source**: https://github.com/svix/svix-webhooks
- **Image**: svix/svix-server:latest
- **Category**: API Management / Webhooks
- **Compose reference**: Official server/docker-compose.yml (adapted, removed build: directive)
- **Services**: svix-server (webhook server), postgres (database), pgbouncer (connection pooler), redis (queue/cache)
- **Ports**: 8071 (API)
