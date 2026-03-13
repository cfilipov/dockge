# WriteFreely
**Project:** https://writefreely.org
**Source:** https://github.com/writefreely/writefreely
**Status:** done
**Compose source:** https://github.com/writefreely/writefreely/blob/develop/docker-compose.prod.yml

## What was done
- Found docker-compose.prod.yml in repo
- Replaced `image: writefreely` with `ghcr.io/writefreely/writefreely:latest` (official ghcr image)
- Replaced linuxserver mariadb with standard mariadb:11
- Created .env with database credentials
- Created data/ and db/ bind mount directories with .gitkeep

## Issues
- None
