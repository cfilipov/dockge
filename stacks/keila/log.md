# Keila Stack

## Sources
- Repository: https://github.com/pentacent/keila
- Compose file: https://raw.githubusercontent.com/pentacent/keila/main/ops/docker-compose.yml

## Notes
Keila is an open-source newsletter tool (alternative to Mailchimp).
The compose file is from the official repository's `ops/` directory.
Removed the `build` context (only relevant for local development) and kept only the image reference.
Removed the postgres port mapping (not needed for inter-service communication).

## Services
- **keila** (`pentacent/keila:latest`) - Newsletter application (port 4000)
- **postgres** (`postgres:alpine`) - PostgreSQL database
