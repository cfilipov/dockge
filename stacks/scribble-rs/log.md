# Scribble.rs

## Source
https://github.com/scribble-rs/scribble.rs

## Description
Scribble.rs is a free, privacy-respecting Pictionary game (alternative to skribbl.io). No ads, no accounts required. Players draw and guess words in real-time.

## Stack Details
- **app**: Scribble.rs Go application (biosmarcel/scribble.rs) on port 8080

## Configuration
- All configuration via environment variables
- `PORT`: Internal HTTP port (default 8080)
- `ROOT_PATH`: URL path prefix for reverse proxy setups
- `CORS_ALLOWED_ORIGINS`: CORS origins (default *)
- `LOBBY_CLEANUP_INTERVAL`: How often idle lobbies are cleaned up (default 90s)

## Notes
- Lightweight Go binary (~7MB image), multi-arch (amd64, arm64, armv7)
- No database required - all state is in-memory
- No persistent volumes needed
