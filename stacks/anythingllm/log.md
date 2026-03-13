# AnythingLLM

## Sources
- Repository: https://github.com/Mintplex-Labs/anything-llm
- Docker docs: https://github.com/Mintplex-Labs/anything-llm/blob/master/docker/HOW_TO_USE_DOCKER.md
- Docker Hub: mintplexlabs/anythingllm

## Notes
- Based on the official docker run command from HOW_TO_USE_DOCKER.md
- The repo's docker-compose.yml uses a build context; converted to use the pre-built Docker Hub image instead
- Image supports amd64 and arm64
- Used named volume instead of host bind mount for portability
- Requires SYS_ADMIN capability
- Access at http://localhost:3001
