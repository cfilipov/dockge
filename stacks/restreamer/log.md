# Restreamer Stack

## Source
- GitHub: https://github.com/datarhei/restreamer
- Category: Media Streaming - Video Streaming

## Description
Restreamer is a complete streaming server solution for self-hosting. It allows video streaming on websites without a streaming provider, supporting RTMP/SRT ingest and HLS/RTMP output with a modern web UI.

## Stack Components
- **restreamer**: datarhei Restreamer (FFmpeg-based streaming core)

## Notes
- Single container with embedded FFmpeg and web interface
- HTTP on 8080, HTTPS on 8181, RTMP ingest on 1935, SRT ingest on 6000/udp
- Admin UI accessible at http://host:8080
- Supports re-streaming to YouTube, Twitch, Facebook, and other platforms
- VAAPI/CUDA variants available for hardware-accelerated transcoding
