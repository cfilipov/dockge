# Sync-in

## Source
- Website: https://sync-in.com
- GitHub: https://github.com/Sync-in/server
- Docker Hub: https://hub.docker.com/r/syncin/server

## Description
Sync-in is a secure, open-source platform for file storage, sharing, collaboration, and syncing. It features real-time collaborative editing, permission management, and desktop/CLI clients. Supports Collabora Online and OnlyOffice integration.

## Stack Components
- **syncin**: Sync-in server (syncin/server:latest)
- **postgres**: PostgreSQL database (postgres:16-alpine)
- **redis**: Redis cache/queue (redis:7-alpine)

## Ports
- 3000: Sync-in web interface

## Volumes
- syncin_data: Application file storage
- postgres_data: Database storage
- redis_data: Redis persistence

## Configuration Notes
- DATABASE_URL connects to the PostgreSQL container
- REDIS_URL connects to the Redis container
- SECRET_KEY must be changed for production use
- SITE_URL should match the public-facing URL
