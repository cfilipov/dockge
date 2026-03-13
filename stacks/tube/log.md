# Tube Stack

## Source
- GitHub: https://github.com/prologic/tube
- Category: Media Streaming - Video Streaming

## Description
Tube is a YouTube-like self-hosted video sharing application. It provides a simple way to host and share video content with automatic transcoding and a clean web UI.

## Stack Components
- **tube**: Tube video sharing server (Go)

## Notes
- Simple single-container setup
- Web UI on port 8000
- Video library stored in /data volume
- No external database required (embedded storage)
- Supports automatic video transcoding via FFmpeg
