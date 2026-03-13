# IFM (Improved File Manager)

## Source
- GitHub: https://github.com/misterunknown/ifm
- Docker Hub: https://hub.docker.com/r/misterunknown/ifm

## Description
Web-based file manager which comes as a single file solution using HTML5, CSS3, JavaScript, and PHP. Based on the official PHP Alpine Docker image.

## Stack
- misterunknown/ifm:latest — main application (port 80)

## Notes
- Single-file PHP file manager
- Set IFM_AUTH=1 to enable authentication
- Files managed in /var/www volume
- Exposes port 80 inside the container
