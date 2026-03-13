# LibreBooking

## Sources
- GitHub: https://github.com/LibreBooking/librebooking
- Docker repo: https://github.com/LibreBooking/docker
- Docker Compose: https://github.com/LibreBooking/docker/blob/master/.examples/docker/docker-compose-local.yml
- Docker Hub: https://hub.docker.com/r/librebooking/librebooking

## Notes
- Compose taken from official docker repo's `.examples/docker/docker-compose-local.yml`
- Three services: MariaDB, app, and cron (background jobs)
- Inlined env_file variables into environment sections for portability
- Uses `linuxserver/mariadb:10.6.13` as specified in official example
- Uses `librebooking/librebooking:4.1.0` as specified in official example
- Cron service runs scheduled tasks (reminders, etc.) using same image with different entrypoint
- Removed `name: librebooking` top-level property (Dockge manages stack names)
- Uses .env file for passwords and timezone
