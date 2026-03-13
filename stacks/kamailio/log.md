# Kamailio Stack

## Research
- GitHub: kamailio/kamailio
- Official Docker image: kamailio/kamailio-ci on Docker Hub
- Kamailio is a high-performance SIP proxy/registrar/router

## Compose
- Uses official `kamailio/kamailio-ci:latest` image
- Exposes SIP ports 5060 (UDP/TCP) and 5061 (TLS)
- Bind mount for configuration directory
- Memory allocation via environment variables
