# PmWiki Stack

## Source
- Docker Hub: sesceu/pmwiki
- Project: https://www.pmwiki.org/

## Description
PmWiki is a wiki-based system for collaborative creation and maintenance of websites. Uses flat files (no database required). This image bundles Apache + PHP with PmWiki pre-installed.

## Volumes
- `pmwiki-data` — wiki page data (wiki.d/)
- `pmwiki-uploads` — uploaded files

## Ports
- 8080 → 80 (HTTP)
