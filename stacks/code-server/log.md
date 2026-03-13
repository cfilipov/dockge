# code-server

## Sources
- GitHub: https://github.com/coder/code-server
- Install docs: https://github.com/coder/code-server/blob/main/docs/install.md
- Docker Hub: codercom/code-server

## Research Notes
- No docker-compose file in the repository
- Docker run command found in install.md documentation
- Image: `codercom/code-server:latest` (supports amd64 and arm64)
- Port: 8080
- Volumes: ~/.config, ~/.local, project directory
- Environment: DOCKER_USER
- Converted docker run command to compose format
