# Snapcast

## Source
- GitHub: https://github.com/badaix/snapcast
- Docker Image: saiyato/snapserver

## Description
Snapcast is a synchronous multiroom audio player. It consists of a server that distributes audio to connected clients, keeping them perfectly in sync. The server reads audio from a named pipe (/tmp/snapfifo).

## Ports
- 1704: Audio streaming port
- 1705: Control port (JSON-RPC)
- 1780: HTTP control/web interface

## Volumes
- data: Server configuration
- /tmp/snapfifo: Named pipe audio input

## Notes
- Not a standalone player - extends existing audio players (MPD, Mopidy, etc.)
- Clients connect and receive perfectly synchronized audio
- Often paired with MPD or Mopidy as the audio source
- Web-based control interface on port 1780
