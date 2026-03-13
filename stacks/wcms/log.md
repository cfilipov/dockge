# W (WCMS) Stack

## Source
- GitHub: vincent-peugnet/wcms
- No official Docker image available

## Description
W is a flat-file CMS/wiki engine written in PHP. Since there is no official Docker image, this stack uses the generic php:apache image. The W source code would need to be downloaded into the volume manually.

## Notes
- No dedicated Docker image exists; using php:8.3-apache as base
- W source must be placed in the wcms-data volume at /var/www/html

## Volumes
- `wcms-data` — W application files and content

## Ports
- 8080 → 80 (HTTP)
