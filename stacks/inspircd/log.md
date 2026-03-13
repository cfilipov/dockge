# InspIRCd

## Sources
- Docker repo: https://github.com/inspircd/inspircd-docker
- Image: inspircd/inspircd-docker
- Main repo references docker repo for container setup

## Notes
- Converted from docker run examples in inspircd-docker README
- Ports: 6667 (plaintext IRC), 6697 (TLS IRC)
- DNSBL disabled by default for local/dev use
- Config volume at /inspircd/conf
