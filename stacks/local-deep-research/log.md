# Local Deep Research

## Sources
- Repository: https://github.com/LearningCircuit/local-deep-research
- Compose file: https://github.com/LearningCircuit/local-deep-research/blob/main/docker-compose.yml

## Notes
- Based on the official docker-compose.yml (CPU-only base config)
- Includes local-deep-research app, Ollama (LLM), and SearXNG (search)
- Removed pinned image digests for cleaner compose file
- Removed custom entrypoint for ollama (requires scripts volume to be pre-populated)
- Simplified healthcheck to use `ollama list` instead of model-specific check
- GPU support available via separate docker-compose.gpu.override.yml in the repo
- Access at http://localhost:5000
- Settings configurable via web UI at http://localhost:5000/settings
