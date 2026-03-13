# Privoxy

## Source
- Website: https://www.privoxy.org
- Docker image: `vimagick/privoxy` (~1.8M pulls on Docker Hub)
- Compose source: https://github.com/vimagick/dockerfiles/tree/master/privoxy

## Research
- No official Docker image; vimagick/privoxy is the most popular community image
- docker-compose.yml found in vimagick/dockerfiles GitHub repo
- Exposes port 8118 for proxy access
- Mounts config, user.action, and user.filter files

## Compose
- Directly adapted from vimagick/dockerfiles docker-compose.yml
- Removed `version: "3.8"` field for Compose V2 compatibility
