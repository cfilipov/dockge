# Apaxy

- **Status**: ok
- **Source**: https://github.com/oupala/apaxy
- **Image**: httpd:2.4-alpine (upstream uses build-only; adapted to stock Apache)
- **Notes**: Apache directory listing theme. Upstream only has a build-based compose file with no published image. This uses stock httpd with autoindex config. Users should clone apaxy theme files into the share volume for full styling.
