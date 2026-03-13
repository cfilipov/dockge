# musikcube

## Source
- GitHub: https://github.com/clangen/musikcube
- Docker Image: linuxserver/musikcube

## Description
musikcube is a cross-platform, terminal-based audio engine, library, player and server written in C++. The server component (musikcubed) provides a streaming audio server that can be accessed remotely.

## Ports
- 7905: Metadata server port
- 7906: Audio streaming port

## Volumes
- music: Music library (read-only)
- data: Application configuration and database

## Notes
- Server component allows remote streaming via musikcube clients
- Primarily a terminal-based application with server capabilities
- Supports MP3, FLAC, OGG, and other common audio formats
