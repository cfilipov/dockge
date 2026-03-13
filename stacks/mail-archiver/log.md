# mail-archiver
**Project:** https://github.com/s1t5/mail-archiver
**Source:** https://github.com/s1t5/mail-archiver
**Status:** done
**Compose source:** docker-compose.yml from GitHub repo (main branch), adapted to use pre-built image
## What was done
- Created compose.yaml with mailarchive-app and postgres services
- Replaced build: with image: s1t5/mailarchiver:latest (Docker Hub)
- Omitted appsettings.json mount (app has defaults)
- Created mount point directories
## Issues
- None
