# Odoo Stack Research Log

## Sources
- Docker Hub (official image): https://hub.docker.com/_/odoo
- Docker library docs: https://github.com/docker-library/docs/blob/master/odoo/README.md
- GitHub: https://github.com/odoo/odoo

## Notes
- Image: `odoo:18` (official Docker library image)
- Database: PostgreSQL 15
- Default port: 8069
- Env vars: HOST, PORT, USER, PASSWORD for DB connection
- Volumes: /var/lib/odoo (data), /mnt/extra-addons (custom addons), /etc/odoo (config)
- Compose derived from docker run examples in official docs
- Access at http://localhost:8069 after startup
