# Review Board

- **Status**: ok
- **Source**: https://github.com/reviewboard/reviewboard
- **Image**: beanbag/reviewboard:7.0
- **Notes**: Code review tool. Compose based on upstream contrib/docker/examples/docker-compose.postgres.yaml. Includes PostgreSQL, memcached, nginx (reverse proxy), and the Review Board app. The init-reviewboard-db.sh script creates the application database user. Optional RabbitMQ and Doc Converter services omitted for simplicity.
