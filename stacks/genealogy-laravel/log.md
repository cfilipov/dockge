# Genealogy (liberu-genealogy) — Research Log

## Result: SKIPPED

The repository has a docker-compose.yml but it uses `build: context: .` for the app, horizon, and scheduler services — requiring the full source tree to build. No pre-built Docker image is published on Docker Hub. Cannot create a compose fixture without a published image.

## Sources Checked
1. https://github.com/liberu-genealogy/genealogy-laravel — README, docker-compose.yml, Dockerfile
2. https://hub.docker.com/r/liberusoftware/genealogy-laravel — 404
3. https://hub.docker.com/u/liberu — no images found
