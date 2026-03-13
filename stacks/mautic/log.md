# Mautic Stack

## Sources
- Repository: https://github.com/mautic/mautic
- Docker repository: https://github.com/mautic/docker-mautic
- Compose file: https://raw.githubusercontent.com/mautic/docker-mautic/main/examples/basic/docker-compose.yml
- Env files: examples/basic/.env and examples/basic/.mautic_env

## Notes
Mautic is an open-source marketing automation platform. The official docker-mautic repo
provides example compose files. This uses the "basic" example with Apache and Doctrine queue.
The original setup uses an env_file (.mautic_env) for Mautic-specific vars; here they are
inlined into the compose environment sections for simplicity (Dockge doesn't support env_file).
The compose network naming was simplified (removed custom network name).

## Services
- **db** (`mysql:lts`) - MySQL database
- **mautic_web** (`mautic/mautic:latest`) - Mautic web interface (port 8080)
- **mautic_cron** (`mautic/mautic:latest`) - Cron job runner
- **mautic_worker** (`mautic/mautic:latest`) - Message queue worker
