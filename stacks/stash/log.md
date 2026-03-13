# Stash

## Overview
Stash is a self-hosted web-based media organizer with tagging, filtering, and DLNA support.

## Image
- `stashapp/stash:latest` (official, Docker Hub)

## Ports
- 9999 → 9999 (web UI)

## Volumes
- `stash_config` — configuration and scrapers
- `stash_metadata` — metadata database
- `stash_cache` — cache
- `stash_blobs` — binary blob data
- `stash_generated` — generated content (screenshots, previews, transcodes)
- Data bind mount for media collection

## Source
- https://github.com/stashapp/stash
- Based on official docker/production/docker-compose.yml
