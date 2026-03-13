# Hi.Events

## Sources
- GitHub: https://github.com/HiEventsDev/hi.events
- Docker Compose: https://github.com/HiEventsDev/hi.events/blob/develop/docker/all-in-one/docker-compose.yml
- Docker Hub: https://hub.docker.com/r/daveearley/hi.events-all-in-one
- .env.example: https://github.com/HiEventsDev/hi.events/blob/develop/docker/all-in-one/.env.example

## Notes
- Based on the all-in-one compose from the official repo (develop branch)
- Replaced `build` directive with pre-built `daveearley/hi.events-all-in-one` image
- Kept postgres:17-alpine and redis:7-alpine as specified in official compose
- Healthchecks preserved from original
- Trimmed environment variables to essential ones (removed Stripe, mail config, etc.)
- Uses .env file for secrets and database credentials
- Mail set to log mode by default
