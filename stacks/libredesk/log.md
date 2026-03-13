# Libredesk

## Source
- GitHub: https://github.com/abhinavxd/libredesk
- Compose: https://github.com/abhinavxd/libredesk/blob/main/docker-compose.yml
- Config sample: https://github.com/abhinavxd/libredesk/blob/main/config.sample.toml

## Research
- Official docker-compose.yml from repo root
- Image: libredesk/libredesk:latest
- Three services: PostgreSQL 17 + Redis 7 + Libredesk app
- Requires config.toml bind-mount (included from config.sample.toml)
- Simplified: removed container_name, custom networks, localhost-only port binds
- Web UI on port 9000
- encryption_key should be changed before production use
