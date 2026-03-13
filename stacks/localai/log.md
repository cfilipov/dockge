# LocalAI

## Sources
- Repository: https://github.com/mudler/LocalAI
- Compose file: https://github.com/mudler/LocalAI/blob/master/docker-compose.yaml

## Notes
- Based on the official docker-compose.yaml from the repository root
- Uses the quay.io image (quay.io/go-skynet/local-ai:master)
- Removed build context, env_file reference, and model command for standalone use
- Removed commented-out PostgreSQL service
- Access at http://localhost:8080
- Models are stored in a named volume; can be loaded via the API
