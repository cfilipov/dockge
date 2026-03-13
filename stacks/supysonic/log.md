# Supysonic

## Source
- GitHub: https://github.com/spl0k/supysonic
- Docker Image: ogarcia/supysonic

## Description
Supysonic is a Python implementation of the Subsonic server API. It allows you to stream your music collection using any Subsonic-compatible client.

## Ports
- 8080: Web interface and Subsonic API

## Volumes
- music: Music library (read-only)
- data: Application database and cache

## Notes
- Compatible with all Subsonic API clients (DSub, Ultrasonic, etc.)
- Supports SQLite (default) or PostgreSQL/MySQL databases
- Lightweight alternative to full Subsonic server
- Library scanning via CLI or web interface
