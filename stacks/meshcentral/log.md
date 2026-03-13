# MeshCentral

## Sources
- Official repo: https://github.com/Ylianst/MeshCentral
- Official compose.yaml: https://raw.githubusercontent.com/Ylianst/MeshCentral/master/docker/compose.yaml
- Docker directory README: https://github.com/Ylianst/MeshCentral/tree/master/docker

## Notes
- MeshCentral is a full computer management web site for remote desktop, terminal, and file access
- Official image: `ghcr.io/ylianst/meshcentral:latest`
- Compose taken directly from the official docker/compose.yaml in the repository
- Hardcoded port values instead of variable substitution from original (PORT=443, REDIR_PORT=80)
- Supports MongoDB, PostgreSQL, MySQL/MariaDB backends (MongoDB shown here)
- Multiple named volumes for data, files, web assets, and backups
- Web UI accessible on port 443 (HTTPS) with redirect from port 80
- Set HOSTNAME to your actual domain for production use
