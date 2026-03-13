# BunkerWeb

- **Category**: Web Servers
- **Source**: https://github.com/bunkerity/bunkerweb
- **Image**: `bunkerity/bunkerweb:latest`
- **Description**: Next-generation Web Application Firewall (WAF) with NGINX under the hood. Includes built-in security features like ModSecurity, bad bot blocking, anti-DDoS, and Let's Encrypt integration.
- **Ports**: 80 (HTTP), 443 (HTTPS)
- **Services**: bunkerweb (main), bw-scheduler, bw-docker-proxy (socket proxy)
- **Notes**: Requires Docker socket access via socket proxy for auto-configuration
