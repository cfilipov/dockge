# Openfire

## Source
- Repository: https://github.com/igniterealtime/Openfire
- Docker image: igniterealtime/openfire
- Docs: https://download.igniterealtime.org/openfire/docs/latest/documentation/docker.html

## Research
- No docker-compose.yml in main repo, but has a Dockerfile
- Official Docker documentation at download.igniterealtime.org has docker run and compose examples
- Image published as igniterealtime/openfire on Docker Hub
- Dockerfile exposes many ports; used the key ones from the official docs example
- Used named volumes for data and logs persistence

## Ports
- 5222: XMPP client connections
- 5269: XMPP server federation
- 7070: HTTP binding
- 7443: HTTPS binding
- 9090: Admin console
