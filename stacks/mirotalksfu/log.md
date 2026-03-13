# MiroTalk SFU

## Source
- Repository: https://github.com/miroslavpejic85/mirotalksfu
- docker-compose.template.yml: https://raw.githubusercontent.com/miroslavpejic85/mirotalksfu/main/docker-compose.template.yml
- .env.template: https://raw.githubusercontent.com/miroslavpejic85/mirotalksfu/main/.env.template

## Images
- mirotalk/sfu:latest

## Notes
- Single-service compose with env file and config.js mounted read-only
- Original template also mounts `./app/src/config.js:/src/app/src/config.js:ro`; omitted here as config.js is large and env vars can override most settings
- Ports: 3010 (web), 40000-40100 (WebRTC media, TCP+UDP)
- SFU_ANNOUNCED_IP must be set to the server's public IP for WebRTC to work externally
- Supports optional recording and RTMP streaming features
