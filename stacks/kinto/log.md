# Kinto Stack

## Source
- GitHub: https://github.com/Kinto/kinto
- Image: kinto/kinto-server

## Services
- **web**: Kinto JSON storage API on port 8888
- **db**: PostgreSQL 14 for storage and permissions
- **cache**: Memcached for caching layer

## Notes
- Based on upstream docker-compose.yml (removed build: directive)
- Uses PostgreSQL for both storage and permission backends
- Memcached provides the caching layer
