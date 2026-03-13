# Exim

## Sources
- Official repo: https://github.com/Exim/exim — no Docker files
- Docker Hub: https://hub.docker.com/r/camptocamp/exim (45k+ pulls)

## Image
- `camptocamp/exim:latest` (community image, well-documented)

## Notes
- No official Docker image from the Exim project
- Used camptocamp/exim which is a well-maintained community image designed for sending email
- Compose derived from Docker Hub README environment variable documentation
- Key env vars: POSTMASTER (required), MAILNAME (required), RELAY_HOST/PORT/USERNAME/PASSWORD (optional)
- Exposes SMTP on port 25
