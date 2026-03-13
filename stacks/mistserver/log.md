# MistServer

## Overview
MistServer is a multi-standard streaming media server supporting RTMP, RTSP, HLS, DASH, WebRTC, and more.

## Image
- `ddvtech/mistserver:latest` (official, Docker Hub)

## Ports
- 4242 — management API/web UI
- 8080 — HTTP streaming
- 1935 — RTMP
- 5554 — RTSP

## Volumes
- `mist_config` — server configuration
- Media bind mount (read-only)

## Source
- https://github.com/DDVTECH/mistserver
