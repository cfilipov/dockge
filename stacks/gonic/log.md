# gonic

- **Source**: https://github.com/sentriz/gonic
- **Docker image**: sentriz/gonic
- **Description**: Lightweight Subsonic-compatible music streaming server written in Go. Supports browsing by folder and tags, transcoding, podcasts, jukebox mode, and last.fm/ListenBrainz scrobbling.
- **Ports**: 4747 (web UI + Subsonic API)
- **Default login**: admin / admin
- **Volumes**: music (read-only), podcasts, playlists, cache, data (database)
