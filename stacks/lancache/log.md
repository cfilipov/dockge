# Lancache

- **Source**: https://github.com/lancachenet/monolithic
- **Category**: Games - Administrative Utilities & Control Panels
- **Description**: LAN cache for game downloads (Steam, Epic, Origin, Battle.net, etc.)
- **Docker image**: lancachenet/monolithic:latest + lancachenet/lancache-dns:latest
- **Compose source**: Official docker-compose repo (lancachenet/docker-compose)
- **Notes**: Caches game downloads on LAN to save bandwidth. DNS service redirects game CDN traffic to local cache. Set LANCACHE_IP and DNS_BIND_IP to your server's LAN IP.
