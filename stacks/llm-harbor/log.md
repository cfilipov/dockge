# LLM Harbor

## Sources
- Repository: https://github.com/av/harbor
- README: https://github.com/av/harbor/blob/main/README.md

## Status: SKIPPED

Harbor is a CLI orchestration tool that dynamically generates and manages Docker Compose files for various LLM services (Ollama, Open WebUI, etc.). It is not itself a Docker-deployable application. The root compose.yml only defines a shared network — actual service configurations are generated at runtime by the `harbor` CLI. No pre-built Docker image exists for Harbor itself.
