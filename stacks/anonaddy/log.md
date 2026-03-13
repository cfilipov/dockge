# AnonAddy Stack

## Source
- GitHub: https://github.com/anonaddy/anonaddy
- Docker repo: https://github.com/anonaddy/docker
- Image: anonaddy/anonaddy:latest

## What was done
- Based compose on official example from anonaddy/docker repo (examples/compose/compose.yml)
- Replaced env_file reference with inline environment variables
- Added .env file with all required substitution variables
- Created data/ and db/ directories for bind mounts
