# Briefkasten
**Project:** https://github.com/ndom91/briefkasten
**Source:** https://github.com/ndom91/briefkasten
**Status:** done
**Compose source:** Based on repo's docker-compose.yml, adapted with ndom91/briefkasten Docker Hub image
## What was done
- Created compose.yaml with briefkasten app + PostgreSQL database
- Used ndom91/briefkasten:latest from Docker Hub instead of build context
- Added .env with default configuration values
- Configured NextAuth environment variables
## Issues
- Repo's original compose uses build: context, replaced with Docker Hub image
