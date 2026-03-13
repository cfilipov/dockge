# Sshwifty

## Sources
- Official repo: https://github.com/nirui/sshwifty
- Docker Hub: https://hub.docker.com/r/niruix/sshwifty
- README docker run example

## Notes
- Sshwifty is a web-based SSH and Telnet client
- Official image: `niruix/sshwifty:latest`
- Compose derived from the docker run command in the official README
- Web UI accessible on port 8182
- TLS can be configured via SSHWIFTY_DOCKER_TLSCERT and SSHWIFTY_DOCKER_TLSCERTKEY environment variables
- Typically deployed behind a reverse proxy for TLS termination
