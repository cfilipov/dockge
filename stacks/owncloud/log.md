# ownCloud Stack

## Source
- GitHub: https://github.com/owncloud/core
- Image: owncloud/server

## Services
- **owncloud**: ownCloud server on port 8080
- **db**: MariaDB 10.11 database
- **redis**: Redis 6 for file locking

## Notes
- Uses owncloud/server Docker image (not core repo which has no compose)
- MariaDB with UTF-8 MB4 support enabled
- Redis enabled for caching and file locking
