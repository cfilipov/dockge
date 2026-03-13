# LibreTime

- **Source**: https://github.com/libretime/libretime
- **Docker images**: ghcr.io/libretime/libretime-{api,legacy,playout,analyzer,worker,nginx}, ghcr.io/libretime/icecast
- **Reference**: https://github.com/libretime/libretime/blob/main/docker-compose.yml
- **Description**: Open-source radio broadcast automation. Multi-service stack with API, legacy web UI, playout/Liquidsoap, analyzer, worker, Nginx, PostgreSQL, RabbitMQ, and Icecast.
- **Ports**: 8080 (web UI), 8000 (Icecast stream), 8001-8002 (Liquidsoap streams)
- **Config**: config.yml bind-mounted into all LibreTime services
