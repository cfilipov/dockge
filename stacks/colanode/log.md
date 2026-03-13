# Colanode

## Overview
Colanode is a collaborative workspace platform. It provides real-time collaboration features with a modern web interface.

## Image
- **GHCR**: `ghcr.io/colanode/server`, `ghcr.io/colanode/web`
- **Source**: https://github.com/colanode/colanode
- **Compose reference**: `hosting/docker/docker-compose.yaml` in the repo

## Stack Details
- PostgreSQL with pgvector extension for data storage
- Valkey (Redis-compatible) for caching/pub-sub
- Server API on port 3000
- Web frontend on port 4000
- Optional MinIO and Mailpit (omitted for simplicity)

## Notes
- Based on official upstream docker-compose.yaml
- Uses pgvector for vector search capabilities
- Valkey replaces Redis as the caching layer
