# CNCjs Stack Research Log

## Sources
- Docker Hub: https://hub.docker.com/r/cncjs/cncjs (image exists: cncjs/cncjs)
- Dockerfile in repo: https://github.com/cncjs/cncjs/blob/master/Dockerfile (exposes port 8000, Node.js app)
- GitHub README: No docker-compose example provided in README

## Notes
- CNCjs is a web-based CNC controller interface
- Port 8000 is the default web UI port (from Dockerfile EXPOSE)
- Privileged mode and device mappings included for USB serial device access (CNC machines connect via /dev/ttyUSB0 or /dev/ttyACM0)
- Data volume for persistent configuration
- No official docker-compose.yml found in repository; compose file constructed from Dockerfile and Docker Hub image
