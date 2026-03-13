# OpenOlitor

## Sources
- Repository: https://github.com/OpenOlitor/openolitor-server
- Docker Compose repo: https://github.com/OpenOlitor/openolitor-docker-compose
- Compose file: https://github.com/OpenOlitor/openolitor-docker-compose/blob/master/docker-compose.yml
- Docker images: openolitor/openolitor-server, openolitor/openolitor-client-admin, openolitor/openolitor-client-kundenportal (Docker Hub)

## Notes
- CSA/Solawi management platform (Scala/Akka backend, Angular clients)
- Multi-service setup: server, admin client, customer portal, nginx reverse proxy, MariaDB, MinIO (S3), PDF converter, mail proxy
- Removed static IP assignments from original compose (not needed for Docker default networking)
- Config files (server conf, DB schema, nginx conf, client configs) are bind-mounted from stack directory
- All config files sourced from the dedicated docker-compose repository
- Removed `version: '2.4'` field for Compose V2 compatibility
- Used named volumes instead of bind-mount data directories
