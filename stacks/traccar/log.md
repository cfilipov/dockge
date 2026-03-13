# Traccar

- **Source**: https://github.com/traccar/traccar
- **Description**: Modern GPS tracking platform supporting 200+ protocols
- **Compose reference**: Official `docker/compose/traccar-mysql.yaml`
- **Services**: traccar (Java GPS server), database (MySQL 8.4)
- **Default ports**: 8082 (web UI), 5000-5500 (device protocols)
- **Notes**: Supports environment-based configuration via CONFIG_USE_ENVIRONMENT_VARIABLES
