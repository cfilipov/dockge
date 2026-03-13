# Prosody IM

## Source
- Project: https://hg.prosody.im/
- Docker repo: https://github.com/prosody/prosody-docker
- Docker image: prosody/prosody (Docker Hub)

## Research
- Non-GitHub project (Mercurial at hg.prosody.im), but has GitHub Docker repo
- prosody/prosody-docker README documents ports, env vars, and volume mounts
- No docker-compose.yml in repo; composed from docker run examples and documentation
- Environment variables LOCAL/DOMAIN/PASSWORD create an admin user on startup

## Ports
- 5222: XMPP client-to-server
- 5269: XMPP server-to-server
- 5280: BOSH/WebSocket
- 5281: Secure BOSH/WebSocket
