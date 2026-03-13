# Lyrion Music Server

- **Source**: https://github.com/lms-community/slimserver
- **Docker image**: lmscommunity/lyrionmusicserver
- **Reference**: https://hub.docker.com/r/lmscommunity/lyrionmusicserver
- **Description**: Streaming audio server (formerly Logitech Media Server / Squeezebox Server). Supports Squeezebox hardware players and software clients. Uses host networking for player discovery.
- **Ports**: 9000 (web UI), 9090 (CLI), 3483 (Slim protocol) - via host networking
- **Volumes**: config, music (read-only), playlists
