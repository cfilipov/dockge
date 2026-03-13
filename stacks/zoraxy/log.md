# Zoraxy

- **Category**: Web Servers / Reverse Proxy
- **Source**: https://github.com/tobychui/zoraxy
- **Image**: `zoraxydocker/zoraxy:latest`
- **Description**: General purpose HTTP reverse proxy and forwarding tool with a web-based management UI. Features include automatic SSL, GeoIP blocking, access control, Docker container discovery, and WebSocket support.
- **Ports**: 80 (HTTP), 443 (HTTPS), 8000 (Admin UI)
- **Volumes**: `./config` for persistent config, `./plugin` for plugins
- **Notes**: Docker socket mounted for container auto-discovery
