# Mox Stack

## Source
- GitHub: https://github.com/mjl-/mox
- Image: r.xmox.nl/mox:latest

## What was done
- Based on official docker-compose.yml from repo
- Uses host networking as required by mox for proper IP handling
- Created config/, data/, www/ directories for bind mounts
- Includes healthcheck monitoring SMTP port 25
