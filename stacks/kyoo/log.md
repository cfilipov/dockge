# Kyoo

## Overview
Kyoo is a self-hosted media browser and streaming platform with automatic metadata fetching, transcoding, and multi-user support.

## Images
- `ghcr.io/zoriya/kyoo_front:edge` — web frontend
- `ghcr.io/zoriya/kyoo_auth:edge` — authentication service
- `ghcr.io/zoriya/kyoo_api:edge` — API backend
- `ghcr.io/zoriya/kyoo_scanner:edge` — media library scanner
- `ghcr.io/zoriya/kyoo_transcoder:edge` — video transcoder
- `postgres:16-alpine` — database

## Ports
- 8901 → 8901 (web UI via front)

## Notes
- Simplified from upstream compose (removed Traefik labels, YAML anchors, hardware accel profiles)
- Hardware acceleration profiles (nvidia, vaapi, qsv) available upstream

## Source
- https://github.com/zoriya/Kyoo
