# Rapidbay Stack

## Source
- GitHub: https://github.com/hauxir/rapidbay
- Category: Media Streaming - Video Streaming

## Description
Rapidbay is a self-hosted torrent video streaming service. It allows searching for and streaming video content directly from torrents in the browser via HTTP.

## Stack Components
- **rapidbay**: Rapidbay torrent streaming application (Python)

## Notes
- Single container setup with torrent client built in
- Web UI on port 5000 for searching and streaming
- Optional Jackett integration for enhanced search capabilities
- Downloads cached in /tmp/rapidbay volume
