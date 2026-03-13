# Pairdrop

- **Source**: https://github.com/schlagmichdoch/pairdrop
- **Category**: File Transfer - Single-click & Drag-n-drop Upload
- **Description**: Local file sharing in your browser, inspired by Apple's AirDrop. Peer-to-peer via WebRTC with websocket fallback.
- **Image(s)**: `lscr.io/linuxserver/pairdrop:latest`
- **Compose source**: Adapted from upstream `docker-compose.yml`
- **Notes**: No persistent storage needed (peer-to-peer transfers). Supports WebRTC with optional websocket fallback. Rate limiting and STUN/TURN configuration available via environment variables.
