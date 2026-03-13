# Firefly III

- **Source**: https://github.com/firefly-iii/firefly-iii
- **Images**: `fireflyiii/core:latest`, `mariadb:lts`, `alpine` (cron)
- **Status**: created
- **Notes**: Based on the official docker-compose from `firefly-iii/docker`. Includes the app, MariaDB database, and a cron container for scheduled tasks. APP_KEY must be exactly 32 characters. STATIC_CRON_TOKEN must also be exactly 32 characters. Uses separate .env instead of the upstream's split .env/.db.env pattern for Dockge compatibility.
