# Dotclear
**Project:** https://dotclear.org
**Source:** https://github.com/dotclear/dotclear
**Status:** done
**Compose source:** https://github.com/JcDenis/docker-dotclear/blob/master/docker-compose.yaml

## What was done
- Found official Docker compose from JcDenis/docker-dotclear (community-maintained, actively updated)
- Created compose.yaml with two services: mariadb + dotclear app
- Created .env with all required variables
- Changed port from 80 to 8080 to avoid conflicts

## Issues
- No official Docker image from dotclear project; using community image jcpd/docker-dotclear
