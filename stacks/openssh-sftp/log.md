# OpenSSH SFTP Server Stack

## Source
- Docker Hub: atmoz/sftp
- GitHub: https://github.com/atmoz/sftp

## Services
- **sftp**: OpenSSH SFTP server on port 2222

## Notes
- Uses atmoz/sftp, the most popular SFTP Docker image
- User credentials passed via command string (user:password:uid)
- Upload directory mounted as named volume
