# ArchivesSpace
**Project:** https://archivesspace.org/
**Source:** https://github.com/archivesspace/archivesspace
**Status:** done
**Compose source:** docker-compose-prod.yml from GitHub repo (master branch)
## What was done
- Created compose.yaml based on the official production docker-compose
- Includes app, db (MySQL 8), solr, and nginx proxy services
- Created .env with default values for required variables
- Created mount point directories (plugins, config, locales, stylesheets, sql, backups, proxy-config)
## Issues
- Omitted db-backup service to keep the stack simpler for testing
