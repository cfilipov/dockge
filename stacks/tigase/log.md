# Tigase

## Source
- Repository: https://github.com/tigase/tigase-server
- Docker repo: https://github.com/tigase/tigase-xmpp-server-docker
- Docker image: tigase/tigase-xmpp-server

## Research
- No docker-compose in main repo; found in dedicated Docker repo
- docker-compose/docker-compose.yml in tigase-xmpp-server-docker repo
- Fixed DB_PORT from 3306 (MySQL) to 5432 to match the postgres image used
- Removed `version:` field for Compose V2 compliance
- Converted bind mounts to named volumes for portability
- Reduced port list to most commonly needed ports

## Ports
- 5222: XMPP client-to-server (StartTLS)
- 5223: XMPP client-to-server (DirectTLS)
- 5269: XMPP server-to-server federation
- 5280: BOSH connections
- 5290: WebSocket
- 8080: HTTP server (web setup, REST API)
