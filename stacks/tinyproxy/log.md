# Tinyproxy

## Source
- GitHub: https://github.com/tinyproxy/tinyproxy
- Docker image: `monokal/tinyproxy:latest` (~810K pulls)
- Docker Hub: https://hub.docker.com/r/monokal/tinyproxy
- Also: `dannydirect/tinyproxy` (~14M pulls, deprecated in favor of monokal)

## Research
- No official Docker image from tinyproxy project
- monokal/tinyproxy is the maintained community image (successor to dannydirect)
- Docker Hub README provides docker run examples
- Port 8888 for proxy access, ACL via command argument
- Supports BASIC_AUTH_USER/PASSWORD env vars and filter file mounting

## Compose
- Converted from docker run command on Docker Hub
- Uses `command: ANY` for unrestricted access (can be changed to IP/CIDR)
- Optional basic auth via environment variables
