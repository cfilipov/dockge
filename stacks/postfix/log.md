# Postfix

## Sources
- Docker Hub: https://hub.docker.com/r/boky/postfix (11.7M pulls)
- Docker Hub README documents all env vars and docker run examples

## Image
- `boky/postfix:latest` (Alpine-based, well-maintained, 11.7M pulls)

## Notes
- No official Docker image from Postfix project (postfix.org)
- Used boky/postfix — popular, lightweight relay-focused Postfix image
- Compose derived from docker run example in Docker Hub README
- Uses port 587 (submission) by default, not port 25
- Key env vars: HOSTNAME, ALLOWED_SENDER_DOMAINS, RELAYHOST, RELAYHOST_USERNAME, RELAYHOST_PASSWORD, MYNETWORKS
- Designed as an outgoing relay for Docker environments, not a full end-user mail server
