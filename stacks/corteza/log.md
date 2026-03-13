# Corteza

## Sources
- GitHub repo: https://github.com/cortezaproject/corteza
- Official deployment docs: https://docs.cortezaproject.org/corteza-docs/2024.9/devops-guide/examples/deploy-offline/index.html
- Docker image: cortezaproject/corteza

## Notes
- Compose adapted from the official offline deployment example in Corteza docs
- Uses Percona 8.0 (MySQL-compatible) as recommended by the docs
- The .env file provides DB_DSN and other config as documented
- Default version pinned to 2024.9 (latest stable at time of writing)
