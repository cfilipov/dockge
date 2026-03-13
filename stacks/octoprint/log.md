# OctoPrint Stack Research Log

## Sources
- Official Docker repo: https://github.com/OctoPrint/octoprint-docker
- docker-compose.yml: https://raw.githubusercontent.com/OctoPrint/octoprint-docker/master/docker-compose.yml
- Image: octoprint/octoprint (Docker Hub)

## Notes
- OctoPrint is a web interface for controlling 3D printers
- Compose based directly on the official docker-compose.yml from octoprint-docker repo
- Removed `version: '2.4'` (not needed in Compose V2)
- Uncommented device mappings and ENABLE_MJPG_STREAMER for webcam support
- Port 80 is the default web UI port
- Named volume `octoprint` for persistent data
- Multi-arch support: arm64, arm/v7, amd64 (great for Raspberry Pi)
