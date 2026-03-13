# SWAG (Secure Web Application Gateway)

- **Category**: Web Servers / Reverse Proxy
- **Source**: https://github.com/linuxserver/docker-swag
- **Image**: `lscr.io/linuxserver/swag:latest`
- **Description**: NGINX-based reverse proxy with built-in Let's Encrypt/ZeroSSL certificate management, fail2ban intrusion prevention, and pre-configured reverse proxy configs for popular apps. By LinuxServer.io.
- **Ports**: 443 (HTTPS), 80 (HTTP)
- **Notes**: Requires NET_ADMIN capability. Configure URL and validation method via environment.
