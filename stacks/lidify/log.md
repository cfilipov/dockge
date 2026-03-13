# Lidify

- **Source**: https://github.com/TheWicklowWolf/Lidify
- **Image**: `thewicklowwolf/lidify:latest`
- **Description**: Music discovery tool providing recommendations based on Lidarr artists. Uses Last.fm API for artist recommendations (Spotify API no longer supported as of Nov 2024).
- **Ports**: 5002 -> 5000
- **Volumes**: config, localtime
- **Key env vars**: lidarr_address, lidarr_api_key, last_fm_api_key, last_fm_api_secret, mode
- **Category**: Media Management
