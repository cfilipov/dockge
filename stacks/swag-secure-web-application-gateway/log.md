# SWAG - Secure Web Application Gateway

- **Status**: ok
- **Source**: https://github.com/linuxserver/docker-swag
- **Image**: lscr.io/linuxserver/swag:latest
- **Notes**: Nginx reverse proxy with built-in Certbot (Let's Encrypt/ZeroSSL) and fail2ban. Requires NET_ADMIN capability. Set STAGING=true for testing. Update URL and VALIDATION for production use. Port 443 required for HTTPS, port 80 optional for HTTP validation.
