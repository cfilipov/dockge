# Teleport

- **Status**: created
- **Source**: https://github.com/gravitational/teleport
- **Image**: public.ecr.aws/gravitational/teleport-distroless:17
- **Notes**: Teleport is an infrastructure access platform. The official image is published to public.ecr.aws (not Docker Hub). No official docker-compose.yml found in the repo; compose file created based on Teleport documentation and standard port mappings. Requires teleport.yaml configuration in the config volume for production use.
