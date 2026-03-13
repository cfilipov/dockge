# Open-Meteo

- **Status**: ok
- **Source**: https://github.com/open-meteo/open-meteo
- **Image**: ghcr.io/open-meteo/open-meteo (GitHub Container Registry)
- **Notes**: Based on upstream docker-compose.yml. Removed build directive. Sync service downloads weather data; API service serves it. Syncs DWD ICON temperature_2m by default.
