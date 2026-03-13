# Lychee

- **Source**: https://github.com/LycheeOrg/Lychee
- **Docker image**: lycheeorg/lychee:latest
- **Compose ref**: https://github.com/LycheeOrg/Lychee-Docker/blob/master/docker-compose.yml
- **Description**: Free photo management with albums, sharing, EXIF data display, and multi-user support
- **Services**: lychee (Laravel app), lychee_db (MariaDB), lychee_cache (Redis)
- **Notes**: Three-service stack. Conf, uploads, sym, and logs directories bind-mounted. Redis used for caching.
