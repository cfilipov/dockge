# NetAlertX

## Status: DONE

## Sources
- https://raw.githubusercontent.com/netalertx/NetAlertX/main/docker-compose.yml — official compose file
- https://github.com/netalertx/NetAlertX/pkgs/container/netalertx — GHCR image

## Notes
- Image: `ghcr.io/netalertx/netalertx:latest`
- Uses host networking for ARP scanning
- Read-only container with tmpfs for /tmp
- Requires NET_ADMIN, NET_RAW capabilities
- Port 20211 (web UI), 20212 (GraphQL API)
- Simplified from upstream compose (removed build directives, hardcoded reasonable defaults)
