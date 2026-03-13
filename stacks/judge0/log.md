# Judge0 CE

## Sources
- GitHub: https://github.com/judge0/judge0
- Official docker-compose.yml: https://github.com/judge0/judge0/blob/master/docker-compose.yml
- Config file: https://github.com/judge0/judge0/blob/master/judge0.conf

## Research Notes
- Official docker-compose.yml found in repository root
- Image: `judge0/judge0:latest`
- Port: 2358 (API)
- Requires PostgreSQL and Redis
- judge0.conf bind-mounted as config file and env_file
- Server and worker are separate containers using same image
- Containers run in privileged mode (needed for code execution sandboxing)
- REDIS_PASSWORD and POSTGRES_PASSWORD must be set before use
- Compose file taken directly from repository
