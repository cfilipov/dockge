# vod2pod-rss

## Source
- GitHub: https://github.com/madiele/vod2pod-rss
- Docker Image: madiele/vod2pod-rss

## Description
Vod2Pod-RSS converts YouTube or Twitch channels into podcasts. It creates a podcast RSS feed that can be listened to in any podcast client. VODs are transcoded to MP3 on the fly with no server storage needed.

## Services
- **vod2pod**: Main Rust application
- **redis**: Redis cache for state management

## Ports
- 8080: Web interface and RSS feeds

## Notes
- Supports YouTube and Twitch as sources
- On-the-fly MP3 transcoding (configurable bitrate)
- Optional API keys for YouTube and Twitch
- Redis required for caching and state
