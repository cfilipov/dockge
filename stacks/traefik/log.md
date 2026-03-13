# Traefik

- **Category**: Web Servers / Reverse Proxy
- **Source**: https://hub.docker.com/_/traefik (Official Docker image)
- **Image**: `traefik:v3.0`
- **Description**: Cloud-native application proxy and load balancer with automatic service discovery via Docker labels. Supports Let's Encrypt, middleware chains, and multiple providers.
- **Ports**: 80 (HTTP), 443 (HTTPS), 8080 (Dashboard)
- **Config files**: `traefik.yml` (static config, bind-mounted), `acme.json` (certificate storage)
- **Notes**: Requires Docker socket access for auto-discovery
