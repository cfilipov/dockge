# SRS (Simple Realtime Server) Stack

## Source
- GitHub: https://github.com/ossrs/srs
- Category: Media Streaming - Video Streaming

## Description
SRS is a simple, high-efficiency, real-time media server supporting RTMP, WebRTC, HLS, HTTP-FLV, SRT, and MPEG-DASH. It is widely used for live streaming and video conferencing.

## Stack Components
- **srs**: SRS media server (C++)

## Notes
- Single container with powerful multi-protocol support
- RTMP ingest (1935), HTTP API (1985), HTTP server/HLS (8080), WebRTC (8000/udp)
- Config file bind-mounted for customization
- CANDIDATE env var must be set to server's public IP for WebRTC
- Supports RTMP-to-WebRTC and WebRTC-to-RTMP transcoding
- SRS v6 is the latest major version
