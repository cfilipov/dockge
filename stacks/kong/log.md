# Kong

- **Source**: https://github.com/Kong/kong
- **Image**: kong:latest
- **Category**: API Management
- **Compose reference**: Official Kong/docker-kong compose (adapted, removed secrets file, removed profile gating)
- **Services**: kong (gateway), kong-migrations (bootstrap), db (postgres)
- **Ports**: 8000 (proxy), 8443 (SSL proxy), 8001 (admin API), 8002 (admin GUI)
