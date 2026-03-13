# Phorge

- **Status**: ok
- **Source**: https://we.phorge.it/ (Phabricator fork)
- **Image**: zeigren/phorge:latest
- **Notes**: Phorge is a community fork of Phabricator. No official Docker image exists. Using zeigren/phorge which is the most maintained community image. Compose based on https://github.com/Zeigren/phorge_docker. Includes Caddy as reverse proxy and MariaDB for the database. Caddyfile config mounted for PHP-FPM proxying.
