# MiroTalk C2C

## Source
- Repository: https://github.com/miroslavpejic85/mirotalkc2c
- docker-compose.template.yml: https://raw.githubusercontent.com/miroslavpejic85/mirotalkc2c/main/docker-compose.template.yml
- .env.template: https://raw.githubusercontent.com/miroslavpejic85/mirotalkc2c/main/.env.template

## Images
- mirotalk/c2c:latest

## Notes
- Simple single-service compose with env file mounted read-only
- App accessible at http://localhost:8080
- .env simplified from template; TURN server disabled by default (requires account)
- Supports OIDC, Mattermost, Sentry integrations (all disabled by default)
