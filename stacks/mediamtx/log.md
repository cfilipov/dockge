# MediaMTX

## Overview
MediaMTX (formerly rtsp-simple-server) is a ready-to-use SRT/WebRTC/RTSP/RTMP/LL-HLS media server and media proxy.

## Image
- `bluenviron/mediamtx:latest` (official, Docker Hub)

## Ports
- 8554 — RTSP
- 1935 — RTMP
- 8888 — HLS
- 8889 — WebRTC
- 8890/udp — SRT
- 9997 — API

## Config
- `mediamtx.yml` — main configuration (bind-mounted)

## Source
- https://github.com/bluenviron/mediamtx
