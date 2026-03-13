# PinePods

## Source
- GitHub: https://github.com/madeofpendletonwool/PinePods
- Docker Image: madeofpendletonwool/pinepods

## Description
PinePods is a Rust-based podcast management system with multi-user support. It manages podcasts with a central database and provides browser-based and mobile clients.

## Services
- **pinepods**: Main application (Rust backend + Yew frontend)
- **db**: PostgreSQL database
- **valkey**: Valkey (Redis-compatible) cache

## Ports
- 8040: Web interface

## Volumes
- pgdata: PostgreSQL data
- downloads: Podcast downloads
- backups: Application backups

## Notes
- Supports Podcast Index and iTunes search
- Built-in gpodder server for external app sync
- Native mobile apps for iOS and Android
- Multi-user with individual settings and subscriptions
