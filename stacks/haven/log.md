# Haven
**Project:** https://havenweb.org
**Source:** https://github.com/havenweb/haven
**Status:** done
**Compose source:** https://github.com/havenweb/haven/blob/master/deploymentscripts/docker-compose.yml

## What was done
- Found standalone docker-compose using ghcr.io/havenweb/haven:latest
- Created compose.yaml with Haven + PostgreSQL
- Created .env with DB password and admin credentials
- Upgraded postgres to 16-alpine, removed non-durability flags

## Issues
- None
