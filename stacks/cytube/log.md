# CyTube Stack

## Source
- GitHub: https://github.com/calzoneman/sync
- Category: Media Streaming - Video Streaming

## Description
CyTube is a web-based synchronized media playback platform with chat. Users can create channels where groups watch videos together in sync.

## Stack Components
- **cytube**: Main CyTube application (Node.js)
- **cytube-db**: MariaDB 11 database

## Notes
- No official Docker image on Docker Hub; using GitHub Container Registry
- Config file (`config.yaml`) is bind-mounted for customization
- Default HTTP port 8080, WebSocket IO port 1443
- Requires MySQL/MariaDB for channel and user data storage
