# Syncthing

## Source
- GitHub: https://github.com/syncthing/syncthing
- Docker Hub: https://hub.docker.com/r/syncthing/syncthing

## Description
Syncthing is a continuous file synchronization program. It synchronizes files between two or more computers in real time, safely protecting data from prying eyes. It is open source, decentralized, and uses TLS for all communication.

## Stack Components
- **syncthing**: Syncthing server (syncthing/syncthing:latest)

## Ports
- 8384: Web GUI
- 22000/tcp: Syncthing protocol (file transfer)
- 22000/udp: Syncthing QUIC protocol
- 21027/udp: Local discovery

## Volumes
- syncthing_data: Synchronized files and configuration

## Configuration Notes
- PUID/PGID control the user/group ID inside the container
- STGUIADDRESS binds the GUI to all interfaces (required for Docker access)
- Healthcheck uses the built-in /rest/noauth/health endpoint
- Based on official Docker documentation from the Syncthing repo
- hostname can be customized via SYNCTHING_HOSTNAME
