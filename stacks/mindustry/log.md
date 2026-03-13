# Mindustry

- **Source**: https://github.com/Anuken/Mindustry
- **Description**: Open-source factory-building/tower-defense game server
- **Image**: hetsh/mindustry:latest
- **Ports**: 6567/tcp + 6567/udp (game server)
- **Notes**: Mindustry is a sandbox tower-defense game. The dedicated server uses port 6567 for both TCP and UDP. The `stdin_open` and `tty` options allow interactive console access for server administration. Several community Docker images exist (hetsh, frankbaele, checker8763); hetsh/mindustry was chosen for its simplicity. Server configuration and maps are stored in the config volume.
