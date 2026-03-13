# Sonarr

- **Source**: https://github.com/Sonarr/Sonarr
- **Image**: lscr.io/linuxserver/sonarr:latest (LinuxServer)
- **Port**: 8989
- **Description**: PVR for Usenet and BitTorrent users. Monitors RSS feeds for new TV episodes, grabs/sorts/renames them, and auto-upgrades quality.
- **Notes**: Uses LinuxServer.io base image with PUID/PGID. Volumes for /tv and /downloads support hardlinks when on same filesystem.
