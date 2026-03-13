# OvenMediaEngine Stack

## Source
- GitHub: https://github.com/AirenSoft/OvenMediaEngine
- Category: Media Streaming - Video Streaming

## Description
OvenMediaEngine (OME) is a sub-second latency live streaming server supporting WebRTC, LLHLS, RTMP, SRT, and more. It provides ultra-low latency streaming capabilities.

## Stack Components
- **origin**: OvenMediaEngine origin server

## Notes
- Based on official docker-compose.yml (simplified to origin-only, no edge)
- Supports multiple ingest protocols: RTMP (1935), SRT (9999)
- Supports multiple output protocols: WebRTC, LLHLS (3333)
- UDP port range 10000-10004 used for WebRTC ICE candidates
- Edge server can be added for scalable distribution
- Set OME_HOST_IP to the server's public IP for WebRTC to work
