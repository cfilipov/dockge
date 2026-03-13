# Fasten Health

## Sources
- Repository: https://github.com/fastenhealth/fasten-onprem
- Compose file: `docker-compose.yml` from main branch (adapted for production image)

## Notes
- Image: `ghcr.io/fastenhealth/fasten-onprem:main`
- Single-service stack (SQLite database stored in volume)
- Removed `build:` block, replaced with pre-built image per repo instructions
- Removed `version:` field for Compose V2 compatibility
- Port 9090 maps to internal 8080
