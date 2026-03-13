# memEx

## Sources
- Official repo: https://codeberg.org/shibao/memEx
- Official docker-compose.yml: https://codeberg.org/shibao/memEx/raw/branch/stable/docker-compose.yml

## Notes
- memEx is a personal knowledge management tool for organizing notes, contexts, and pipelines
- Official image: `shibaobun/memex`
- Compose adapted from the official docker-compose.yml (removed `version:` field, fixed quoted env values)
- Requires PostgreSQL 13
- App listens on port 4000 (exposed, not published — designed for reverse proxy)
- First registered user becomes admin
- SECRET_KEY_BASE and HOST should be configured for production use
