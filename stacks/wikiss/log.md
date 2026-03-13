# WiKiss Stack

## Source
- Project: http://music.music.free.fr/wikiss/
- No official Docker image available

## Description
WiKiss is a minimalist flat-file wiki written in PHP. No database required — stores content as text files. Since there is no official Docker image, this stack uses the generic php:apache image. The WiKiss source code would need to be downloaded into the volume manually.

## Notes
- No dedicated Docker image exists; using php:8.3-apache as base
- WiKiss source must be placed in the wikiss-data volume at /var/www/html

## Volumes
- `wikiss-data` — WiKiss application and content files

## Ports
- 8080 → 80 (HTTP)
