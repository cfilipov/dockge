# openSIPS Stack

## Research
- GitHub: OpenSIPS/opensips
- Official Docker images on Docker Hub: opensips/opensips, opensips/opensips-cp
- OpenSIPS is a multi-functional SIP server (proxy, registrar, router)

## Compose
- Uses official `opensips/opensips:latest` image
- Includes OpenSIPS Control Panel (`opensips/opensips-cp`) on port 8080
- Exposes SIP ports 5060 (UDP/TCP) and 5061 (TLS)
- Bind mount for configuration
