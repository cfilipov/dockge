# Tiny Tiny RSS

- **Source**: https://github.com/tt-rss/tt-rss
- **Images**: cthulhoo/ttrss-fpm-pgsql-static:latest, cthulhoo/ttrss-web-nginx:latest, postgres:15-alpine
- Feature-rich web-based RSS/Atom feed reader. PHP + PostgreSQL.
- Four-container setup: database, app (PHP-FPM), updater (background feed fetcher), web (nginx).
- Based on tt-rss official Docker Compose deployment pattern.
