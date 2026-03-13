# Invidious Stack

## Source
- GitHub: https://github.com/iv-org/invidious
- Category: Media Streaming - Video Streaming

## Description
Invidious is an alternative front-end to YouTube. It provides a privacy-respecting interface without ads, tracking, or JavaScript requirements.

## Stack Components
- **invidious**: Main Invidious application (Crystal)
- **invidious-db**: PostgreSQL 14 database

## Notes
- Official image hosted on Quay.io (not Docker Hub)
- Configuration passed via INVIDIOUS_CONFIG environment variable (YAML block)
- HMAC key should be changed from default for production use
- Database schema is auto-created via `check_tables: true`
