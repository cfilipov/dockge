# Navidrome Music Server

## Source
- GitHub: https://github.com/navidrome/navidrome
- Docker Image: deluan/navidrome

## Description
Navidrome is an open source web-based music collection server and streamer. It gives you freedom to listen to your music collection from any browser or mobile device. Compatible with Subsonic/Airsonic clients.

## Ports
- 4533: Web interface and API

## Volumes
- data: Database and cache
- music: Music library (read-only)

## Notes
- Subsonic API compatible - works with DSub, Ultrasonic, play:Sub, etc.
- Automatic library scanning on configurable schedule
- Supports transcoding, smart playlists, and multi-user
- First user created becomes admin
