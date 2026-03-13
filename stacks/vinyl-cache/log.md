# Vinyl Cache (Varnish)

- **Category**: Web Servers / Cache
- **Source**: https://github.com/wodby/varnish (Docker image by Wodby)
- **Image**: `wodby/varnish:6`
- **Description**: Varnish HTTP cache accelerator packaged by Wodby. Acts as a reverse caching proxy to speed up web applications by caching responses in memory.
- **Ports**: 6081 (Varnish)
- **Services**: varnish (cache), backend (nginx for demo)
- **Notes**: Configure backend host/port via environment variables. Includes nginx backend for demonstration.
