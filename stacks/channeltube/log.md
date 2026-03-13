# ChannelTube

- **Source**: https://github.com/TheWicklowWolf/ChannelTube
- **Image**: `thewicklowwolf/channeltube:latest`
- **Description**: YouTube channel sync tool using yt-dlp. Downloads video and audio from subscribed channels on a schedule.
- **Ports**: 5100 -> 5000
- **Volumes**: config, video downloads, audio downloads, localtime
- **Key env vars**: PUID, PGID, thread_limit, video_format_id, audio_format_id
- **Category**: Media Management
