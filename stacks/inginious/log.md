# INGInious

- **Source**: https://github.com/INGInious/INGInious
- **Image**: inginious/frontend:latest, inginious/backend:latest (Docker Hub)
- **Description**: Intelligent grader for programming assignments. Runs student code in isolated Docker containers for automated grading.
- **Services**: frontend (web UI), backend (grading orchestrator), mongodb
- **Compose reference**: Adapted from upstream docker-compose.yml (removed build: directives, kept published images)
- **Notes**: Requires Docker socket mount for the backend to spawn grading containers.
