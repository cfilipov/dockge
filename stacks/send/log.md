# Send

## Status: DONE

## Sources
- https://github.com/timvisee/send - Main repository (fork of Firefox Send)
- https://github.com/timvisee/send/blob/master/docs/docker.md - Docker documentation
- Image: registry.gitlab.com/timvisee/send:latest

## Notes
- WebUI on port 1443
- Requires Redis for metadata storage
- File storage defaults to /uploads (local filesystem)
- Also supports S3 and GCS backends via environment variables
- Compose based on docker.md quickstart and docker-compose.yml in repo
