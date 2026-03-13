# XWiki Stack

## Source
- Docker Hub: xwiki (official image)
- Project: https://www.xwiki.org/

## Description
XWiki is a powerful open-source enterprise wiki platform written in Java. Features structured data, scripting, access control, and extensibility via plugins. Runs on Tomcat with PostgreSQL backend.

## Services
- **xwiki** — XWiki application (Tomcat)
- **db** — PostgreSQL database

## Volumes
- `xwiki-data` — XWiki application data and extensions
- `xwiki-db` — PostgreSQL data

## Ports
- 8080 → 8080 (HTTP)
