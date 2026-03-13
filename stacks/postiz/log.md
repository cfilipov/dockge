# Postiz
**Project:** https://github.com/gitroomhq/postiz-app
**Source:** https://github.com/gitroomhq/postiz-app
**Status:** done
**Compose source:** docker-compose.yaml from repository root

## What was done
- Created compose.yaml based on official docker-compose.yaml
- Includes Postiz app, PostgreSQL 17, Redis 7.2
- Includes Temporal workflow engine with its own PostgreSQL and Elasticsearch
- Uses ghcr.io/gitroomhq/postiz-app:latest image
- Created .env with database password and JWT secret

## Issues
- None
