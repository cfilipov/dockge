# FusionPBX Stack

## Research
- GitHub: fusionpbx/fusionpbx
- No official Docker image or Dockerfile in repo
- Community image `kinsamanka/fusionpbx` (Alpine-based, minimal)
- FusionPBX is a web GUI for FreeSWITCH, typically uses PostgreSQL

## Compose
- Uses `kinsamanka/fusionpbx:latest` with PostgreSQL 15
- Exposes HTTP/HTTPS, SIP (5060/5080), and RTP media ports
- PostgreSQL credentials in .env file
- Named volumes for data persistence
