# Asterisk

## Sources
- GitHub: https://github.com/mlan/docker-asterisk
- Docker Hub: https://hub.docker.com/r/mlan/asterisk
- The official asterisk/asterisk repo does not provide Docker images; mlan/asterisk is a well-maintained community image.

## Compose origin
Based on the demo docker-compose.yml from the mlan/docker-asterisk repository README.

## Notes
- RTP port range limited to 10000-10099 to reduce Docker proxy overhead (default is 10000-20000)
- Supports SIP over UDP (5060), TCP (5060), and TLS (5061)
- WebSMS interface on port 8080
- Built-in intrusion detection (AutoBan) using nftables requires net_admin and net_raw capabilities
