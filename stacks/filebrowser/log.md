# File Browser

## Source
- GitHub: https://github.com/filebrowser/filebrowser
- Docker Hub: https://hub.docker.com/r/filebrowser/filebrowser

## Description
Web-based file managing interface within a specified directory. Supports file uploading, deleting, previewing, renaming, editing, and sharing.

## Stack
- filebrowser/filebrowser:latest — main application (port 80)

## Notes
- Default credentials: admin / admin
- Configuration via filebrowser.json mounted to /.filebrowser.json
- Database stored in /database volume
- Files served from /srv volume
