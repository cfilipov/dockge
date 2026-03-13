# bewCloud Stack

## Source
- GitHub: https://github.com/bewcloud/bewcloud
- Image: ghcr.io/bewcloud/bewcloud

## Services
- **website**: bewCloud app on port 8000
- **postgresql**: PostgreSQL 18 database
- **radicale**: CardDAV/CalDAV server

## Notes
- Based on upstream docker-compose.yml
- Radicale provides CardDAV/CalDAV sync support
- Data files stored in bind mount `./data-files`
