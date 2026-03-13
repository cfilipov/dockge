# Radarr

- **Source**: https://github.com/Radarr/Radarr
- **Image**: lscr.io/linuxserver/radarr:latest (LinuxServer)
- **Port**: 7878
- **Description**: Movie collection manager for Usenet and BitTorrent users. Monitors RSS feeds, grabs/sorts/renames movies, and auto-upgrades quality.
- **Notes**: Uses LinuxServer.io base image with PUID/PGID user mapping. Volumes for /movies and /downloads support hardlinks when on same filesystem.
