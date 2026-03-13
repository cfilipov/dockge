# Personal Management System

- **Source**: https://github.com/Volmarg/personal-management-system
- **Image**: volmarg/personal-management-system:latest + mariadb:11.5.2 + yobasystems/alpine-nginx:stable + adminer:4.8.0
- **Category**: Personal Dashboards
- **Port**: 8002 → 80 (nginx), 8081 → 8080 (adminer)
- **Notes**: Based on upstream docker-compose.yml. Replaced build: directives with pre-built images. Multi-container stack: MariaDB, Nginx, PHP-FPM app, and Adminer. Nginx config provided for PHP-FPM proxying.
