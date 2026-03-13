# chasquid

## Sources
- GitHub repo: https://github.com/albertito/chasquid
- Docker directory: https://github.com/albertito/chasquid/tree/master/docker
- Docker README: https://github.com/albertito/chasquid/blob/master/docker/README.md

## Image
- Official: `registry.gitlab.com/albertito/chasquid:main` (also on Docker Hub as `albertito/chasquid`)

## Notes
- Compose derived from `docker run` command in docker/README.md
- Uses host networking so chasquid can see source IP addresses
- AUTO_CERTS env var triggers automatic Let's Encrypt certificate provisioning
- Data volume stores domains, users, and certificates
- Exposes SMTP (25, 465, 587), IMAP (993), POP3 (995), Sieve (4190), HTTP/S (80, 443) via host network
- Container bundles chasquid + dovecot + certbot via supervisord
