# Squid

## Source
- Website: https://code.launchpad.net/squid
- Docker image: `ubuntu/squid:5.2-22.04_beta` (Canonical official, ~37M pulls)
- Docker Hub: https://hub.docker.com/r/ubuntu/squid

## Research
- Docker Hub description provides docker run command and parameter docs
- `docker run -d --name squid-container -e TZ=UTC -p 3128:3128 ubuntu/squid:5.2-22.04_beta`
- Supports volume mounts for logs, cache data, and config files
- Port 3128 is the standard Squid proxy port

## Compose
- Converted from docker run command on Docker Hub
- Includes log and cache data volumes
- Uses Canonical's official Ubuntu-based image
