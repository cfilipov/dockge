# AnyCable
**Project:** https://anycable.io/
**Source:** https://github.com/anycable/anycable-go
**Status:** done
**Compose source:** Constructed from Docker Hub image and project documentation

## What was done
- Created compose.yaml with anycable-go server and Redis dependency
- Used official anycable/anycable-go image from Docker Hub
- Configured standard environment variables for Redis and RPC connections

## Issues
- The anycable-go repo was archived in Dec 2024, moved to anycable/anycable monorepo
- No official docker-compose.yml found in repo; composed from documentation
