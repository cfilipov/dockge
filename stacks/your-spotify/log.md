# Your Spotify

- **Source**: https://github.com/Yooooomi/your_spotify
- **Image**: yooooomi/your_spotify_server:latest + yooooomi/your_spotify_client:latest + mongo:7
- **Category**: Personal Dashboards
- **Port**: 8080 → 8080 (API), 3000 → 3000 (client)
- **Notes**: No upstream compose file found at expected paths. Compose based on official documentation. Three-container stack: server API, client frontend, MongoDB. Requires Spotify API credentials (SPOTIFY_PUBLIC and SPOTIFY_SECRET).
