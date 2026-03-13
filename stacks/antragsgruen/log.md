# motion.tools (Antragsgruen)

## Sources
- Main repository: https://github.com/CatoTH/antragsgruen
- Docker image repository: https://github.com/devops-ansible/docker-antragsgruen
- Compose file: https://github.com/devops-ansible/docker-antragsgruen/blob/master/docker-compose.yml
- Docker image: devopsansiblede/antragsgruen (Docker Hub)

## Notes
- The main CatoTH/antragsgruen repo only has a development compose file (docker-compose.development.yml) that uses `build:`
- Production Docker setup is maintained separately by devops-ansible/docker-antragsgruen
- Simple two-service setup: MariaDB + Antragsgruen (Apache-based PHP app)
- Removed `version: '3'` field for Compose V2 compatibility
- App accessible on port 8080
