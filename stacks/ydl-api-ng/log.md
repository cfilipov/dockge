# ydl_api_ng

- **Source**: https://github.com/Totonyus/ydl_api_ng
- **Image**: totonyus/ydl_api_ng:latest
- **Port**: 5011 (mapped to container 80)
- **Description**: REST API wrapper around yt-dlp with configurable presets, templates, and download locations. Supports Redis for job queuing.
- **Notes**: Config via params/params.ini. Includes Redis sidecar (can be disabled via DISABLE_REDIS=true). Presets for audio, video at various qualities.
