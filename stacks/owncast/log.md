# Owncast Stack

## Source
- GitHub: https://github.com/owncast/owncast
- Category: Media Streaming - Video Streaming

## Description
Owncast is a self-hosted, open-source live streaming and chat server. It provides a drop-in replacement for platforms like Twitch, with RTMP ingest and HLS output.

## Stack Components
- **owncast**: Owncast streaming server (Go)

## Notes
- Simple single-container setup with embedded SQLite database
- RTMP ingest on port 1935, web UI/HLS on port 8080
- Admin panel at /admin (default password set via environment variable)
- Data persisted in /app/data volume (database, transcoding segments, logs)
- No external database required
