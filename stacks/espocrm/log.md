# EspoCRM

## Sources
- GitHub repo: https://github.com/espocrm/espocrm
- Official Docker docs: https://github.com/espocrm/documentation/blob/master/docs/administration/docker/installation.md
- Docker image: espocrm/espocrm

## Notes
- Compose taken verbatim from official EspoCRM Docker installation documentation
- Includes 4 services: database (MariaDB), main app, daemon (cron jobs), and websocket
- All EspoCRM containers share the same volume for the application files
- Default admin credentials: admin / password
- Access at http://localhost:8080
