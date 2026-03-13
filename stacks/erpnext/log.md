# ERPNext Stack Research Log

## Sources
- Main repo: https://github.com/frappe/erpnext (points to frappe_docker for Docker setup)
- Docker repo: https://github.com/frappe/frappe_docker
- Reference compose: https://github.com/frappe/frappe_docker/blob/main/pwd.yml
- Docker Hub: https://hub.docker.com/r/frappe/erpnext

## Notes
- ERPNext uses the frappe_docker repo for all Docker deployment
- pwd.yml is the "quick disposable demo" single compose file
- compose.yaml is for production (requires more setup)
- Image: `frappe/erpnext:v16.9.0`
- Complex multi-service setup: backend, frontend (nginx), websocket, scheduler, workers, redis, mariadb
- Configurator and create-site are init containers that run once
- Default admin credentials: Administrator / admin
- Replaced deploy.restart_policy with restart: unless-stopped for Compose V2 compatibility
