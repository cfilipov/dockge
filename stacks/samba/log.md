# Samba

## Source
- Docker Hub: https://hub.docker.com/r/dperson/samba

## Description
Samba provides SMB/CIFS file sharing services, enabling file and print sharing with Windows, macOS, and Linux clients. The dperson/samba image is a popular community Docker image for easy Samba deployment.

## Stack Components
- **samba**: Samba file server (dperson/samba:latest)

## Ports
- 445: SMB direct hosting
- 139: NetBIOS session service

## Volumes
- samba_share: Shared file storage

## Configuration Notes
- Users and shares are configured via command-line arguments to the container
- The -u flag creates a user: "username;password"
- The -s flag creates a share: "name;path;browsable;readonly;guest;users;admins;comment"
- The -p flag disables minimum SMB protocol version restrictions
- USERID/GROUPID set the file ownership UID/GID
