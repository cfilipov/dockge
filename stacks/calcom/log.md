# Cal.com

## Sources
- GitHub: https://github.com/calcom/cal.com
- Docker repo (archived): https://github.com/calcom/docker
- Docker Compose: https://raw.githubusercontent.com/calcom/docker/main/docker-compose.yaml
- Docker Hub: https://hub.docker.com/r/calcom/cal.com

## Notes
- Original compose from calcom/docker repo (now archived, moved to main monorepo)
- Simplified to use pre-built `calcom/cal.com` image instead of build context
- Removed calcom-api and studio services (optional/advanced)
- Removed redis (optional for basic setup)
- Removed build args (using pre-built image)
- Upgraded postgres from unversioned to postgres:16
- Added healthcheck to postgres
- Uses .env file for variable substitution (secrets, DB credentials)
- Removed `version:` field for Compose V2 compatibility
