# AWStats
**Project:** https://www.awstats.org/
**Source:** https://www.awstats.org/#DEMO
**Status:** done
**Compose source:** Community Docker image pabra/awstats from Docker Hub

## What was done
- Created compose.yaml using the most popular community AWStats Docker image (pabra/awstats)
- Configured with bind mount for log files and persistent data volume
- Created logs/ directory with .gitkeep for bind mount

## Issues
- No official Docker image; used community image pabra/awstats (7 stars on Docker Hub)
