# Druid
**Project:** https://github.com/apache/druid
**Source:** https://github.com/apache/druid
**Status:** done
**Compose source:** distribution/docker/docker-compose.yml from repository

## What was done
- Created compose.yaml based on official Docker Compose from the repo
- Includes PostgreSQL, ZooKeeper, and all 5 Druid services (coordinator, broker, historical, middlemanager, router)
- Created environment file with Druid configuration
- Uses official apache/druid:31.0.1 image

## Issues
- None
