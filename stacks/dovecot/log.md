# Dovecot - Docker Research Log

## Result: DONE

## Research Steps

1. Checked repo root (dovecot/core on GitHub) - no Dockerfile or docker-compose in repo
2. Found official Docker docs at https://doc.dovecot.org/2.4.2/installation/docker.html
3. Official image: `dovecot/dovecot:latest` on Docker Hub
4. Converted docker run examples from official docs to compose format

## Source

- Official Docker docs: https://doc.dovecot.org/2.4.2/installation/docker.html
- Docker Hub: https://hub.docker.com/r/dovecot/dovecot
- Image: `dovecot/dovecot:latest`

## Ports

- 31143: IMAP
- 31993: IMAPS
- 31110: POP3
- 31990: POP3S
- 31465: Submissions
- 31587: Submission
- 31024: LMTPS
- 34190: ManageSieve
- 8080: HTTP API
- 9090: Metrics

## Notes

- Rootless since v2.4.0, runs as vmail user (UID 1000)
- Non-privileged ports (31xxx) since v2.4.1
- Mail storage at /srv/vmail, config drop-ins at /etc/dovecot/conf.d
- Default protocols: IMAP, Submission, LMTP, Sieve
- POP3 requires additional config in drop-in files
- TLS certs can be mounted at /etc/dovecot/ssl (tls.crt, tls.key)
