# Frappe Helpdesk

## Source
- GitHub: https://github.com/frappe/helpdesk
- Compose: https://github.com/frappe/helpdesk/blob/main/docker/docker-compose.yml

## Research
- Official docker-compose.yml found in docker/ directory of the repo
- Uses frappe/bench:latest image with MariaDB 10.8 and Redis
- Init script bootstraps Frappe bench, installs helpdesk app, creates site
- Default credentials: Administrator / admin
- Web UI on port 8000 at http://helpdesk.localhost:8000/helpdesk
- Development-oriented setup (no separate production Docker image)
