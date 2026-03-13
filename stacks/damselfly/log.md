# Damselfly

- **Source**: https://github.com/webreaper/damselfly
- **Docker image**: webreaper/damselfly:latest
- **Compose ref**: Upstream compose uses build context; adapted for prebuilt image
- **Description**: Digital asset management for photos with AI face detection, object recognition, and keyword tagging
- **Services**: damselfly (Blazor Server app with embedded SQLite)
- **Notes**: Default port 6363. No external database required — uses SQLite stored in /config.
