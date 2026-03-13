# OpenSMTPD

## Sources
- Official project: https://www.opensmtpd.org/
- Docker image repo: https://github.com/wodby/opensmtpd
- Docker Hub: https://hub.docker.com/r/wodby/opensmtpd (1.85M pulls)

## Image
- `wodby/opensmtpd:latest` (Alpine-based, multi-arch, most popular OpenSMTPD image)

## Notes
- No official Docker image from the OpenSMTPD project
- Used wodby/opensmtpd which is well-maintained and heavily used (1.85M pulls)
- Compose derived from environment variable documentation in GitHub README
- Key env vars: RELAY_HOST, RELAY_PORT, RELAY_PROTO, RELAY_USER, RELAY_PASSWORD
- Supports Docker secrets via _FILE suffix env vars
- Exposes SMTP on port 25
