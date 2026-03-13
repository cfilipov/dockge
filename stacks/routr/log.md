# Routr Stack

## Research
- GitHub: fonoster/routr
- Official Docker image: fonoster/routr-one on Docker Hub
- Compose example found directly in README
- Routr is a lightweight SIP proxy with a modern API

## Compose
- Uses official `fonoster/routr-one:latest` image (all-in-one with embedded PostgreSQL)
- Exposes API port 51908 and SIP port 5060 (UDP)
- EXTERNAL_ADDRS env var required for SIP NAT traversal
- Named volume for PostgreSQL data persistence
