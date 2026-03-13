# Islandora Stack

## Source
- Repository: https://github.com/Islandora/islandora (Drupal module)
- Docker deployment: https://github.com/Islandora-Devops/isle-site-template
- Previous Docker project (archived): https://github.com/Islandora-Devops/isle-dc

## Research
- Core Islandora repo is a Drupal module with no Docker files
- Official Docker deployment is via isle-site-template (successor to isle-dc, archived Jan 2026)
- isle-site-template has a 333-line docker-compose.yml with 16 services, secrets files, and cert management
- Simplified to core services, replaced secrets files with environment variables
- Used `islandora/sandbox` image for Drupal (pre-configured Islandora instance)
- Removed traefik, mergepdf, crayfits, and cert management for simplicity
- All images from `islandora/` namespace on Docker Hub with `ISLANDORA_TAG` versioning

## Services
- **drupal**: Islandora Drupal application (port 80)
- **fcrepo**: Fedora 6 repository for digital objects
- **mariadb**: Database backend
- **solr**: Search and indexing
- **activemq**: Message broker
- **alpaca**: Islandora middleware
- **blazegraph**: RDF triplestore
- **cantaloupe**: IIIF image server
- **fits**: File format identification
- **homarus**: FFmpeg derivative service
- **houdini**: ImageMagick derivative service
- **hypercube**: OCR derivative service
- **milliner**: Fedora indexing service

## Notes
- Production deployments should use the full isle-site-template with secrets files and traefik
- The sandbox image includes a pre-built Drupal installation with Islandora modules
- This compose is simplified from the official 16-service production configuration
