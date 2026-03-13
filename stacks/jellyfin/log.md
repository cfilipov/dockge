# Jellyfin

## Overview
Jellyfin is a free and open-source media server and suite of multimedia applications for organizing, managing, and streaming digital media.

## Image
- `jellyfin/jellyfin:latest` (official, Docker Hub)

## Ports
- 8096 → 8096 (HTTP web UI)
- 8920 → 8920 (HTTPS)

## Volumes
- `jellyfin_config` — server configuration and database
- `jellyfin_cache` — transcoding cache
- Media bind mount (read-only)

## Source
- https://github.com/jellyfin/jellyfin
