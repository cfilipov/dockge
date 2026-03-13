# Homepage (by gethomepage)

- **Source**: https://github.com/gethomepage/homepage
- **Image**: ghcr.io/gethomepage/homepage:latest
- **Category**: Personal Dashboards
- **Port**: 3000 → 3000
- **Notes**: No upstream compose file found. Compose based on official documentation. Config directory bind-mounted at /app/config with settings.yaml, services.yaml, bookmarks.yaml, and widgets.yaml. Docker socket mounted read-only for service discovery.
