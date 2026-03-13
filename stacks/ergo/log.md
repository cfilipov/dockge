# Ergo

## Sources
- Official docker-compose.yml: https://github.com/ergochat/ergo/blob/stable/distrib/docker/docker-compose.yml
- Image: ghcr.io/ergochat/ergo:stable

## Notes
- Adapted from official compose file (removed Swarm-specific deploy section, added restart policy)
- Ports: 6667 (plaintext IRC), 6697 (TLS IRC)
- Data stored in named volume mounted at /ircd
