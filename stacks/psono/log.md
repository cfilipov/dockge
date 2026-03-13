# Psono

## Sources
- GitHub mirror: https://github.com/psono/psono-server
- Official docs: https://doc.psono.com
- Quickstart repo: https://gitlab.com/esaqa/psono/psono-quickstart
- Docker Hub: https://hub.docker.com/r/psono/psono-combo

## Notes
- Compose derived from the official quickstart install script (install.sh) on GitLab
- Using Community Edition image (psono/psono-combo:latest); Enterprise uses psono/psono-combo-enterprise:latest
- 4 services: nginx proxy, PostgreSQL 15, psono-combo (server+client), psono-fileserver
- Removed watchtower service (auto-updater, not part of Psono itself)
- Removed quickstart demo user creation commands from psono-combo entrypoint
- Removed legacy `version` field and `links` directives for Compose V2 compatibility
- Removed container_name directives for flexibility
- Requires manual configuration: settings.yaml, config.json, nginx.conf, SSL certificates
