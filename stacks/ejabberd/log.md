# ejabberd

## Source
- Repository: https://github.com/processone/ejabberd
- Docker image: ghcr.io/processone/ejabberd

## Research
- Found CONTAINER.md in repo root with multiple Docker Compose examples
- Used the "Customized Example" as the base, adapted with named volume instead of bind mount for database
- Removed `version:` field for Compose V2 compliance
- Removed bind-mounted config file (not included) to keep it self-contained

## Ports
- 5222: XMPP client connections
- 5269: XMPP server-to-server
- 5280: HTTP admin interface
- 5443: HTTPS/WSS
