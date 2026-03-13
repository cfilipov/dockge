# MiroTalk P2P

## Source
- Repository: https://github.com/miroslavpejic85/mirotalk
- docker-compose.template.yml: https://raw.githubusercontent.com/miroslavpejic85/mirotalk/master/docker-compose.template.yml
- .env.template: https://raw.githubusercontent.com/miroslavpejic85/mirotalk/master/.env.template

## Images
- mirotalk/p2p:latest

## Notes
- Single-service compose with env file mounted read-only
- App accessible at http://localhost:3000
- .env simplified from template; TURN disabled by default
- Supports optional Traefik integration (commented out in original)
