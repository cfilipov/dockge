# Coder

## Sources
- GitHub: https://github.com/coder/coder
- Official compose.yaml: https://github.com/coder/coder/blob/main/compose.yaml
- Docker install docs: https://coder.com/docs/install/docker

## Research Notes
- Official compose.yaml found in repository root
- Image: `ghcr.io/coder/coder:latest`
- Requires PostgreSQL 13+ (compose uses postgres:17)
- Port: 7080
- Needs Docker socket mount for workspace provisioning
- CODER_ACCESS_URL must be set to externally reachable address
- Compose file adapted from official with minor cleanup
