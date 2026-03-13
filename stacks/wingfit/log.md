# Wingfit

## Sources
- Repository: https://github.com/itskovacs/wingfit
- Compose file: `docker-compose.yml` from main branch
- Docker image from README: `ghcr.io/itskovacs/wingfit:5`

## Notes
- Image: `ghcr.io/itskovacs/wingfit:5` (GitHub Container Registry)
- Single-service stack (FastAPI app with SQLite storage)
- Replaced `build: .` with pre-built image per docker run instructions in README
- Removed `127.0.0.1:` binding prefix for broader accessibility
- Port 8080 maps to internal 8000
- Data persisted in `./storage` volume
