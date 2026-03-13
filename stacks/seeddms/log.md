# SeedDMS

## What was done
- Based on bibbox/app-seeddms docker-compose.yml
- Removed external network dependency (bibbox-default-network)
- Services: seeddms (PHP app), MariaDB
- Images: bibbox/seeddms:6.0.19, mariadb:10.11
- Port: 8065 -> 80
- Credentials extracted to .env
