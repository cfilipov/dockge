# PeerTube Stack

## Source
- GitHub: https://github.com/Chocobozzz/PeerTube
- Category: Media Streaming - Video Streaming

## Description
PeerTube is a free, decentralized, and federated video platform powered by ActivityPub and WebTorrent. It allows hosting videos on your own server while federating with other PeerTube instances.

## Stack Components
- **peertube**: Main PeerTube application (Node.js)
- **peertube-db**: PostgreSQL 17 database
- **peertube-redis**: Redis for caching and job queue

## Notes
- Simplified from official production docker-compose (removed nginx, certbot, postfix)
- Exposes PeerTube directly on port 9000 (suitable for dev/testing or behind a reverse proxy)
- RTMP port 1935 for live streaming support
- Official compose includes nginx reverse proxy and certbot for production SSL
- Federation requires a publicly accessible domain with HTTPS
