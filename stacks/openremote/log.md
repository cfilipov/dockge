# OpenRemote

## Source
- GitHub: https://github.com/openremote/openremote
- Docker Hub: openremote/*

## Description
Open-source IoT platform for asset management, data visualization, and automation. Includes protocol agents, rules engine, and multi-tenant support via Keycloak.

## Stack
- **proxy**: HAProxy reverse proxy with Let's Encrypt SSL
- **postgresql**: Database backend
- **keycloak**: Identity and access management
- **manager**: Core OpenRemote manager application

## Notes
- Compose based on official docker-compose.yml from the OpenRemote repo
- Multi-service architecture with health check dependencies
