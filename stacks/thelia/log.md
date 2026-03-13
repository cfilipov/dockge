# Thelia Stack

## Research
- GitHub: thelia/thelia - PHP e-commerce framework
- Found docker-compose.yml in master branch: MariaDB, nginx, php-fpm, encore (build-based)
- No official Docker image
- Simplified: removed encore (build), replaced build-based php-fpm with `php:8.2-fpm`
- Added nginx.conf since upstream mounts one

## Compose
- Based on upstream docker-compose.yml
- Nginx + PHP-FPM + MariaDB
- Created minimal nginx.conf for PHP-FPM proxying
- MariaDB 10.11 (upgraded from upstream's 10.3)
