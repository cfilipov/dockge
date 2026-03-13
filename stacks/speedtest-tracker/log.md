# Speedtest Tracker

## Status: DONE

## Sources
- https://raw.githubusercontent.com/linuxserver/docker-speedtest-tracker/main/README.md — official LinuxServer compose
- https://docs.speedtest-tracker.dev/getting-started/installation — installation docs

## Notes
- Image: `lscr.io/linuxserver/speedtest-tracker:latest` (LinuxServer.io)
- Port 80 (web UI)
- SQLite by default (supports PostgreSQL/MySQL via env vars)
- Simplified from upstream (removed optional env vars with empty values)
