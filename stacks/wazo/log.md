# Wazo Stack

## Research
- GitHub org: wazo-platform (193 repos)
- Docker compose repo: wazo-platform/wazo-docker (experimental)
- Official images on Docker Hub: wazoplatform/wazo-auth, wazoplatform/wazo-calld, wazoplatform/wazo-confd, etc.
- Wazo is a UC (Unified Communications) platform built on Asterisk
- Full setup has 15+ microservices; simplified here to core services

## Compose
- Simplified from official wazo-docker compose (which has 15+ services and build steps)
- Core services: nginx, postgres, rabbitmq, auth, calld, confd
- Uses official wazoplatform/* Docker Hub images
- HTTPS access on port 8443
- PostgreSQL 15 for data, RabbitMQ for messaging
