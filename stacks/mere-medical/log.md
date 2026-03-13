# Mere Medical

## Sources
- Repository: https://github.com/cfu288/mere-medical
- Compose file: `docker-compose.yaml` from main branch

## Notes
- Image: `cfu288/mere-medical:latest` (Docker Hub)
- Single-service stack (offline-first web app, data stored in browser)
- Simplified from upstream: removed docs and demo services, kept only app service
- Reduced environment variables to most common provider integrations
- Port 4200 maps to internal 80 (nginx)
