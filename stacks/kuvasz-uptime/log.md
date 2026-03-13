# Kuvasz Uptime

## Sources
- Official repo: https://github.com/kuvasz-uptime/kuvasz
- Official docs deployment guide: https://kuvasz-uptime.dev/setup/installation/
- Docker Hub: https://hub.docker.com/r/kuvaszmonitoring/kuvasz

## Notes
- Kuvasz is a self-hosted uptime monitor with a PostgreSQL backend
- Official image: `kuvaszmonitoring/kuvasz:latest`
- Requires PostgreSQL 14+ (included via pgautoupgrade image)
- Config file mounted at `/config/kuvasz.yml`
- Compose taken directly from the official deployment guide
- ADMIN_PASSWORD must be >= 12 chars, ADMIN_API_KEY >= 16 chars
