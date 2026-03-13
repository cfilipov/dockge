# Seatsurfing

## Sources
- GitHub: https://github.com/seatsurfing/seatsurfing
- Docker Compose: from README.md in the repo
- Container Registry: ghcr.io/seatsurfing/backend

## Notes
- Compose taken directly from the official README
- Image `ghcr.io/seatsurfing/backend` is the official GHCR image (includes both backend and UI)
- Added healthcheck to postgres service
- Added depends_on with condition for proper startup ordering
- Replaced hardcoded passwords with .env variable substitution
- CRYPT_KEY must be 32 bytes long
- Booking UI at :8080/ui/search/, admin UI at :8080/ui/admin/
- PostgreSQL 17 as specified in official docs
