# Mixpost
**Project:** https://github.com/inovector/MixpostApp
**Source:** https://github.com/inovector/MixpostApp
**Status:** done
**Compose source:** Official docs at docs.mixpost.app/lite/installation/docker

## What was done
- Created compose.yaml based on official Docker documentation (without SSL variant)
- Uses inovector/mixpost:latest image
- Includes MySQL 8.0 and Redis
- Created .env with required variables

## Issues
- APP_KEY needs to be generated via mixpost.app/tools before real use
