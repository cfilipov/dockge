# NGINX

- **Category**: Web Servers
- **Source**: https://hub.docker.com/_/nginx (Official Docker image)
- **Image**: `nginx:latest`
- **Description**: High-performance HTTP server and reverse proxy, as well as an IMAP/POP3 proxy server. Known for its stability, rich feature set, simple configuration, and low resource consumption.
- **Ports**: 8080 -> 80 (HTTP)
- **Config files**: `nginx.conf` (bind-mounted)
- **Volumes**: `./html` for web content
