# Gatus

## Sources
- Official repo: https://github.com/TwiN/gatus
- README docker run command: `docker run -p 8080:8080 --name gatus ghcr.io/twin/gatus:stable`

## Notes
- Gatus is an automated developer-oriented status page with health checks
- Official images: `ghcr.io/twin/gatus:stable` (GHCR) and `twinproduction/gatus:stable` (Docker Hub)
- Configuration via `config.yaml` mounted at `/config`
- Compose derived from the official docker run command in the README
- Also available: Docker Hub image `twinproduction/gatus:stable`
