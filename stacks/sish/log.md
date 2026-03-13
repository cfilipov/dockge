# sish

## Source
- GitHub: https://github.com/antoniomika/sish
- Docker image: `antoniomika/sish:latest`
- Docs: https://docs.ssi.sh/getting-started

## Research
- Official deploy/docker-compose.yml in repo includes letsencrypt sidecar
- docker run example in docs shows standalone usage
- Uses host networking for SSH/HTTP/HTTPS tunneling
- Mounts SSL certs, SSH keys, and public keys directories

## Compose
- Simplified from official deploy/docker-compose.yml (removed letsencrypt sidecar)
- Based on docker run example from docs
- Uses host networking as recommended
- Users need to create ssl/, keys/, pubkeys/ directories and populate them
