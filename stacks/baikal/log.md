# Baïkal

## Sources
- Official site Docker docs: https://sabre.io/baikal/docker-install/
- Community Docker image repo: https://github.com/ckulka/baikal-docker
- Docker Compose example: https://github.com/ckulka/baikal-docker/blob/master/examples/docker-compose.yaml
- Docker Hub: https://hub.docker.com/r/ckulka/baikal

## Notes
- The official Baikal repo (sabre-io/Baikal) does not contain Docker files
- Community-maintained image `ckulka/baikal` is referenced from official docs
- Using the nginx variant as shown in the official example
- Compose file taken directly from the examples directory in ckulka/baikal-docker
- Removed `version: "2"` field for Compose V2 compatibility
