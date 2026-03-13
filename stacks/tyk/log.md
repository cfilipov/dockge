# Tyk

- **Source**: https://github.com/TykTechnologies/tyk
- **Image**: tykio/tyk-gateway:latest
- **Category**: API Management
- **Compose reference**: Official docker-compose.yml uses includes and internal build; recreated for standalone use
- **Services**: tyk-gateway (API gateway), redis (storage)
- **Ports**: 8080 (API gateway)
- **Config**: tyk.conf (gateway config), apps/ (API definitions directory)
