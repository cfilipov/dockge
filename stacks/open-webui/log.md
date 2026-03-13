# Open-WebUI

## Sources
- Repository: https://github.com/open-webui/open-webui
- Compose file: https://github.com/open-webui/open-webui/blob/main/docker-compose.yaml

## Notes
- Based on the official docker-compose.yaml from the repository root
- Includes Open-WebUI frontend and bundled Ollama backend
- Removed build context, using pre-built GHCR image directly
- Removed variable substitution for image tags and port (hardcoded to latest/main and 3000)
- Access at http://localhost:3000
- First user to sign up becomes admin
