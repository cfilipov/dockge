# admidio

## Sources
- Docker Compose example from official README-Docker.md: https://github.com/Admidio/admidio/blob/master/README-Docker.md
- Docker image: admidio/admidio on Docker Hub
- Dockerfile in repo root confirms port 8080, volumes, and environment variables

## Notes
- Official compose uses MariaDB (lts-noble tag)
- Three volumes for persistent files, plugins, and themes
- Environment variables documented in README-Docker.md
- Removed `security_opt: seccomp:unconfined` from official example (not needed for most deployments)
- Changed image tag from `branch_v4.3` to `latest` for broader compatibility
