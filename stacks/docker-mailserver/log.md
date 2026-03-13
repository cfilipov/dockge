# docker-mailserver Stack

## Source
- GitHub: https://github.com/docker-mailserver/docker-mailserver
- Image: ghcr.io/docker-mailserver/docker-mailserver:latest

## What was done
- Based compose on official compose.yaml from the repo root
- Replaced env_file with inline environment variables for common settings
- Created docker-data/dms/ subdirectories for bind mounts
