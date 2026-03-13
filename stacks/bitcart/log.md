# Bitcart

- **Source**: https://github.com/bitcart/bitcart
- **Image**: `bitcart/bitcart:latest` (Docker Hub, 16K+ pulls)
- **Status**: created
- **Notes**: Bitcart uses a generator script (`bitcart-docker`) to produce compose files dynamically. This is a simplified compose based on the core components (API backend, PostgreSQL, Redis). The full deployment with store frontend, admin panel, and crypto daemons requires running their setup script. This fixture covers the essential backend services.
