# Frappe HR (HRMS)

## Sources
- Docker Compose from official repo: https://github.com/frappe/hrms/tree/develop/docker/docker-compose.yml
- Frappe Docker project for production reference: https://github.com/frappe/frappe_docker

## Notes
- Frappe HR is a Frappe framework app, not a standalone Docker image
- No dedicated `frappe/hrms` image on Docker Hub
- The repo provides a development Docker Compose in `docker/docker-compose.yml` using `frappe/bench:latest`
- For production, users deploy via frappe_docker with HRMS installed as a custom app
- Services: MariaDB 10.8, Redis (alpine), Frappe bench
- Default access at http://localhost:8000 after initialization
- Default credentials: Administrator / admin
