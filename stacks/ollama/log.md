# Ollama

## Sources
- Repository: https://github.com/ollama/ollama
- Docker Hub: https://hub.docker.com/r/ollama/ollama

## Notes
- Based on the official Docker Hub image ollama/ollama
- No docker-compose.yml in the repository; created from the standard docker run pattern
- Standard docker run: `docker run -d -v ollama:/root/.ollama -p 11434:11434 ollama/ollama`
- API accessible at http://localhost:11434
- Models stored in /root/.ollama, persisted via named volume
- GPU support available with `--gpus=all` flag (not included by default)
