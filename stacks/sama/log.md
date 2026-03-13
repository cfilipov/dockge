# SAMA
**Project:** https://samacloud.io/
**Source:** https://github.com/SAMA-Communications/sama-server
**Status:** done
**Compose source:** Simplified from official docker-compose-full.yml

## What was done
- Created compose.yaml with sama-server, MongoDB, and Redis (core services)
- Simplified from the full 12-service stack to essential services

## Issues
- Full stack has many services (nginx, push daemon, dashboards, S3); simplified to core
