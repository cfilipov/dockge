# GoDoxy

- **Category**: Web Servers / Reverse Proxy
- **Source**: https://github.com/yusing/godoxy
- **Image**: `ghcr.io/yusing/godoxy:latest`
- **Description**: Easy-to-use Docker-aware reverse proxy with auto-discovery, health checks, custom error pages, and WebSocket support. Consists of a proxy app, frontend UI, and Docker socket proxy.
- **Ports**: Host network mode for proxy, 2375 for socket proxy
- **Services**: socket-proxy, frontend (UI), app (proxy engine)
- **Notes**: Uses network_mode: host for the main proxy. Derived from compose.example.yml upstream.
