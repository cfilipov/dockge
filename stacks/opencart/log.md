# OpenCart Stack

## Research
- Found docker-compose.yml in opencart/opencart GitHub repo
- Original uses `build:` for apache and php services with custom Dockerfiles
- Has many optional services (postgres, redis, memcached, adminer) via profiles
- Core services: Apache, PHP-FPM 8.4, MariaDB

## Changes from upstream
- Replaced `build:` services with Bitnami images (apache, php-fpm)
- Removed profile-based optional services (postgres, redis, memcached, adminer)
- Removed container_name directives, deploy resource limits, and logging config
- Kept MariaDB as the primary database
- All ${VAR} references have defaults and are defined in .env
