# DavMail - Docker Research Log

## Result: DONE

## Research Steps

1. Checked repo root for docker-compose files - not in root
2. GitHub README mentions Docker images on GHCR, points to `src/docker/README.md`
3. Found `src/docker/compose.yml` in the repo - used as basis for compose.yaml
4. Found `src/docker/Dockerfile` - multi-stage build from Debian 12
5. Found `src/docker/entrypoint.sh` - auto-creates config from template if missing

## Source

- Compose file: https://github.com/mguessan/davmail/blob/master/src/docker/compose.yml
- Docker README: https://github.com/mguessan/davmail/blob/master/src/docker/README.md
- Image: `ghcr.io/mguessan/davmail:latest`

## Ports

- 1025: SMTP
- 1143: IMAP
- 1080: CalDAV
- 1389: LDAP

## Notes

- Config is stored in `./config/davmail.properties` (auto-created from template on first run)
- OAuth2 tokens required for Exchange Online (basic auth no longer supported)
- Ports bound to 127.0.0.1 by default to prevent public exposure
