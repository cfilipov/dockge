# multi-scrobbler

## Source
- GitHub: https://github.com/FoxxMD/multi-scrobbler
- Docker Image: foxxmd/multi-scrobbler

## Description
A javascript app to scrobble music you listen to, from multiple sources to multiple clients. Supports Maloja, Last.fm, ListenBrainz, and more.

## Ports
- 9078: Web UI and API

## Volumes
- config: Configuration files for sources and clients

## Notes
- Supports many sources: Spotify, Plex, Jellyfin, Tautulli, Subsonic, etc.
- Supports many scrobble clients: Maloja, Last.fm, ListenBrainz
- Configuration via JSON files in the config directory
