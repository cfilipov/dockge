# InvoicePlane

- **Source**: https://github.com/InvoicePlane/InvoicePlane
- **Compose reference**: `docker-compose.yml` from repo (master branch, uses `build:` directives)
- **Status**: ok
- **Services**: invoiceplane (PHP app), db (MariaDB)
- **Notes**: The repo's compose uses `build:` for php-fpm, nginx, and mariadb. Replaced with the official `invoiceplane/invoiceplane` Docker Hub image (last updated 2015, but available) and a standard MariaDB image. The Docker Hub image bundles PHP + Apache, so nginx is not needed separately.
