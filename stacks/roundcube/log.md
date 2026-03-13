# Roundcube Stack

## Sources
- Official Docker examples repo: https://github.com/roundcube/roundcubemail-docker/blob/master/examples/docker-compose-mysql.yaml
- Docker Hub image: https://hub.docker.com/r/roundcube/roundcubemail

## Changes from upstream
- Removed `version: '2'` field (Compose V2 format)
- Removed `container_name` directives (let Compose manage names)
- Removed deprecated `links` directive (not needed with Compose networking)
- ROUNDCUBEMAIL_DEFAULT_HOST and ROUNDCUBEMAIL_SMTP_SERVER use example.org placeholder - must be configured for actual mail server

## Services
- **roundcubemail**: Roundcube webmail client (Apache + PHP)
- **roundcubedb**: MySQL database for Roundcube data

## Notes
- Access at http://localhost:9001
- Configure ROUNDCUBEMAIL_DEFAULT_HOST and ROUNDCUBEMAIL_SMTP_SERVER to point to your IMAP/SMTP server
- A simpler SQLite-based setup is also available (see upstream examples)
