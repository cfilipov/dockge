# Mainsail Stack Research Log

## Sources
- Official docs: https://docs.mainsail.xyz/setup/docker (docker-compose example provided)
- GitHub: https://github.com/mainsail-crew/mainsail
- Image: ghcr.io/mainsail-crew/mainsail:latest

## Notes
- Mainsail is a Klipper 3D printer web interface (similar to Fluidd)
- Compose file taken directly from official documentation
- config.json bind-mount for configuration (connects to Moonraker API)
- Port 8080 maps to container port 80 (nginx)
