# SnappyMail Stack

## Sources
- Docker Compose from official repo: https://github.com/the-djmaze/snappymail/blob/master/docker-compose.yml
- Docker Hub image: https://hub.docker.com/r/djmaze/snappymail

## Changes from upstream
- Replaced `build:` directive with pre-built `djmaze/snappymail:latest` image
- Removed step-ca and docker-mailserver services (demo/dev infrastructure)
- Removed cert volumes and custom entrypoints (not needed with pre-built image)
- Removed `version: '3.0'` field (Compose V2 format)
- Used named volumes instead of bind mounts for data persistence

## Services
- **snappymail**: SnappyMail webmail client (PHP-FPM + built-in web server)
- **db**: MySQL 5.7 database

## Notes
- Access at http://localhost:8888
- Admin panel at http://localhost:8888/?admin
- Get admin password: `docker exec -it <container> cat /var/lib/snappymail/_data_/_default_/admin_password.txt`
- Configure IMAP/SMTP server settings through the admin panel
