# uWave

## Overview
uWave is a self-hosted collaborative listening platform where users take turns playing music/videos from YouTube and SoundCloud.

## Image
- `imxdev/uwave:latest` from Docker Hub (community image)
- `mongo:7` — MongoDB database
- `redis:7-alpine` — Redis for real-time state

## Ports
- 6042 → 6042 (web UI)

## Volumes
- `mongo_data` — MongoDB data
- `redis_data` — Redis persistence

## Source
- https://github.com/u-wave
