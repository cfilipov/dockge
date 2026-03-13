# BTCPay Server

- **Source**: https://github.com/btcpayserver/btcpayserver
- **Image**: `btcpayserver/btcpayserver:latest` (Docker Hub)
- **Status**: created
- **Notes**: BTCPay Server normally uses a docker-fragment generator for complex multi-service deployments (with Bitcoin/Lightning nodes, Tor, nginx, etc.). This is a simplified compose with just BTCPay Server and PostgreSQL. For full deployment with crypto daemons, use the official btcpayserver-docker setup script.
