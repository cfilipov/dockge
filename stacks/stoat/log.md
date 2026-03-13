# Stoat
**Project:** https://stoat.chat/
**Source:** https://github.com/stoatchat/self-hosted
**Status:** done
**Compose source:** Official compose.yml from self-hosted repo (simplified)

## What was done
- Created compose.yaml based on official compose.yml
- Includes core services: api, events, web, caddy, mongo, redis, rabbitmq, minio, livekit

## Issues
- Simplified from full config; production use requires generate_config.sh for secrets
