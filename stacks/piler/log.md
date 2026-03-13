# Piler
**Project:** https://github.com/jsuto/piler
**Source:** https://github.com/jsuto/piler
**Status:** partial
**Compose source:** Based on repo Dockerfile (docker/Dockerfile) and project requirements
## What was done
- Created compose.yaml with piler, mysql, and memcached services
- No pre-built Docker image exists on Docker Hub or ghcr.io
- Used ubuntu:24.04 as placeholder (matches Dockerfile base)
- Piler requires MySQL and memcached as dependencies
## Issues
- No official pre-built Docker image; ubuntu:24.04 is a placeholder
- Would need to build from source Dockerfile for real deployment
