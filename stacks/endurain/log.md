# Endurain

## Sources
- Repository: https://github.com/endurain-project/endurain
- Compose file: `docker-compose.yml.example` from master branch
- Env file: `.env.example` from master branch

## Notes
- Image: `ghcr.io/endurain-project/endurain:latest`
- Requires PostgreSQL 17.5
- Volume paths changed from `<local_path>/endurain/...` to relative `./...` for Dockge compatibility
- Environment variables configured via `.env` file (shared between app and postgres)
