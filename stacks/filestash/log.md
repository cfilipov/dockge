# Filestash

## Source
- GitHub: https://github.com/mickael-kerjean/filestash
- Docker Hub: https://hub.docker.com/r/machines/filestash

## Description
A modern web client for SFTP, S3, FTP, WebDAV, Git, Minio, LDAP, CalDAV, CardDAV, MySQL, Backblaze and more. Turns cloud storage into a web-based file manager.

## Stack
- machines/filestash:latest — main application (port 8334)

## Notes
- Official compose also includes Collabora Online (wopi_server) for document editing — omitted here for simplicity
- State persisted in /app/data/state/ volume
- Supports multiple storage backends configured via web UI
- Set APPLICATION_URL if running behind a reverse proxy
