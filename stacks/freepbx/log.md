# FreePBX Stack

## Research
- No official GitHub repo with Docker support
- Well-known community image: tiredofit/freepbx on Docker Hub
- Requires MariaDB backend
- Repo is archived but image still available

## Compose
- Uses `tiredofit/freepbx:latest` with MariaDB 10.11
- Exposes HTTP (80/443), SIP (5060), and RTP media ports (18000-18100)
- Environment variables for DB connection in .env file
- Named volumes for data persistence
