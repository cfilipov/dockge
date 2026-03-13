# PodFetch

- **Source**: https://github.com/SamTV12345/PodFetch
- **Image**: `samuel19982/podfetch:latest`
- **Description**: Self-hosted podcast manager written in Rust. Download podcasts and listen online with GPodder integration. Uses SQLite for storage.
- **Ports**: 8000 -> 8000
- **Volumes**: podfetch-podcasts (named), podfetch-db (named)
- **Key env vars**: POLLING_INTERVAL, SERVER_URL, DATABASE_URL
- **Category**: Media Management
