# Snikket

## Source
- Repository: https://github.com/snikket-im/snikket-server
- Docker images: snikket/snikket-server, snikket/snikket-web-proxy, snikket/snikket-cert-manager, snikket/snikket-web-portal

## Research
- docker-compose.yml found in repo root (master branch)
- snikket.conf.example found in repo root with required env vars
- Compose uses env_file: snikket.conf, so included that config file
- Removed `version:` field for Compose V2 compliance
- Multi-service setup: XMPP server, web proxy, cert manager, web portal
- All services use host networking

## Services
- snikket_server: Main XMPP server
- snikket_proxy: Web reverse proxy
- snikket_certs: ACME certificate manager
- snikket_portal: Web admin portal
