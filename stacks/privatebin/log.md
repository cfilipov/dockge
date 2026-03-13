# PrivateBin

- **Source**: https://github.com/PrivateBin/PrivateBin
- **Image**: privatebin/nginx-fpm-alpine:latest
- **Description**: Minimalist zero-knowledge pastebin. Data encrypted/decrypted in browser using 256-bit AES-GCM.
- **Port**: 8080
- **Notes**: Uses read-only container with tmpfs mounts for security. Data stored in /srv/data volume.
