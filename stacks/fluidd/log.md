# Fluidd Stack Research Log

## Sources
- Official docs: https://docs.fluidd.xyz/installation/docker
- GitHub: https://github.com/fluidd-core/fluidd
- Image: ghcr.io/fluidd-core/fluidd (standard, port 80) or ghcr.io/fluidd-core/fluidd-unprivileged (port 8080)

## Notes
- Fluidd is a Klipper 3D printer web interface (frontend only)
- Standard image serves on port 80, unprivileged on port 8080
- PORT env var can override the default serving port
- No docker-compose example provided in official docs; compose constructed from documented image and port info
- Fluidd is a static web frontend that connects to a Moonraker API backend (not included in this compose)
