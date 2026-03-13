# Caddy

- **Category**: Web Servers
- **Source**: https://github.com/caddyserver/caddy
- **Image**: `caddy:2`
- **Description**: Fast, multi-platform web server with automatic HTTPS. Written in Go. Known for its simplicity and Caddyfile configuration format.
- **Ports**: 80 (HTTP), 443 (HTTPS + HTTP/3 via UDP)
- **Volumes**: `./Caddyfile` for config, `./site` for content, named volumes for data/config
- **Config files**: `Caddyfile` (bind-mounted)
