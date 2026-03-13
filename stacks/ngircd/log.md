# ngircd

## Sources
- LinuxServer.io Docker image: https://github.com/linuxserver/docker-ngircd
- Image: lscr.io/linuxserver/ngircd:latest
- Compose example from LinuxServer README

## Notes
- Uses LinuxServer.io image (official ngircd repo has Dockerfile but no published image)
- Port 6667 for IRC connections
- Config volume at /config (generates default ngircd.conf on first run)
- PUID/PGID for file permission mapping
