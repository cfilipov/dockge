# piqueserver

- **Source**: https://github.com/piqueserver/piqueserver (fork of openspades server)
- **Related**: https://github.com/yvt/openspades (client)
- **Description**: Ace of Spades / OpenSpades game server (Python)
- **Image**: piqueserver/piqueserver:latest (tags: latest, master)
- **Ports**: 32887/tcp+udp (game), 32886/tcp (status), 32885/tcp (query)
- **Notes**: piqueserver is a Python-based server for Ace of Spades-compatible clients (including OpenSpades). The official Docker image is available on Docker Hub. Configuration is stored in the /config volume. The server exposes three ports: the main game port (32887), a status port (32886), and a query port (32885).
