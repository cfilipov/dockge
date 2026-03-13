# Cloudreve Stack

## Source
- GitHub: https://github.com/cloudreve/Cloudreve
- Image: cloudreve/cloudreve

## Services
- **cloudreve**: Cloud file management on port 5212
- **postgresql**: PostgreSQL 17 database
- **redis**: Redis cache

## Notes
- Based on upstream docker-compose.yml
- Supports BitTorrent on port 6888 (TCP+UDP)
- PostgreSQL uses trust auth (no password) by default
