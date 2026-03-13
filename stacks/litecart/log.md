# Litecart Stack

## Research
- No docker-compose.yml in the repo, but Docker Hub has `shurco/litecart` image
- README documents `docker run` commands with volume mounts
- Litecart uses embedded SQLite — single container, no external DB needed
- Volumes: lc_base (database/config), lc_digitals (digital products), lc_uploads (uploads), site (website files)

## Changes from upstream
- Created compose.yaml from documented `docker run` instructions
- Used named volumes instead of bind mounts for cleaner management
