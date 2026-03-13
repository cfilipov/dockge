# Galette

- **Source**: https://github.com/galette/galette
- **Images**: `galette/galette:1.1.4` (Docker Hub, 5K+ pulls), `mariadb:10.11`
- **Status**: created
- **Notes**: Based on the official galette-community/docker compose for galette-and-mariadb. After starting, complete the web installer at http://localhost:8080/installer.php. Uses named volumes instead of host bind-mounts (converted from the upstream's host path pattern). Database connection details are entered during the web installer.
