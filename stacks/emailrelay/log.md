# EmailRelay

## Sources
- Docker Hub: https://hub.docker.com/r/dcagatay/emailrelay
- GitHub: https://github.com/dogukancagatay/docker-emailrelay
- docker-compose.yml from repo: https://raw.githubusercontent.com/dogukancagatay/docker-emailrelay/master/docker-compose.yml

## Image
- `dcagatay/emailrelay:latest` (Alpine-based, 25k+ pulls)

## Notes
- Compose taken directly from the project's docker-compose.yml (removed `build: ./`)
- Paired with mailpit as a test mail catcher (web UI on port 8025)
- EmailRelay forwards mail to mailpit on port 1025
- Exposes SMTP on port 25
- Env vars: DEFAULT_OPTS, PORT, SPOOL_DIR, SWAKS_OPTS
