# HomeBox (SysAdminsMedia)

## Source
- Repository: https://github.com/sysadminsmedia/homebox
- Docker image: ghcr.io/sysadminsmedia/homebox:latest

## Compose Source
- Derived from the official docker run command in the GitHub README
- Image and port mapping confirmed from repository documentation

## Notes
- The repo's docker-compose.yml is for development (builds from source)
- Production deployment uses the GHCR image
- SQLite-based, no external database needed
- Internal port 7745, mapped to 3100

## Services
- **homebox**: Home inventory management app (port 3100 -> 7745)
