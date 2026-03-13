# OpenTrashmail

## Sources
- GitHub repo: https://github.com/HaschekSolutions/opentrashmail
- docker-compose.yml: https://raw.githubusercontent.com/HaschekSolutions/opentrashmail/master/docker-compose.yml

## Image
- `hascheksolutions/opentrashmail:1` (official image)

## Notes
- Compose taken directly from the project's docker-compose.yml (removed `version` field)
- Disposable email service with web UI
- SMTP on port 2525 (mapped to container 25), web UI on port 8080
- Data and logs persisted via bind mounts
- Configurable via env vars: DOMAINS, DISCARD_UNKNOWN, ADMIN_ENABLED, PASSWORD, etc.
- Supports TLS, webhooks, IP whitelisting, and admin panel
