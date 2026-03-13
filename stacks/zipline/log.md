# Zipline Stack

## Source
https://github.com/diced/zipline

## Description
ShareX/file upload server with a modern Next.js dashboard. Supports image/file uploads, URL shortening, text paste, themes, and multi-user management with PostgreSQL backend.

## Services
- **zipline** - Main application (ghcr.io/diced/zipline:latest) on port 3000
- **zipline-db** - PostgreSQL 16 database with healthcheck

## Reference
Compose based on upstream `docker-compose.yml` from trunk branch. Includes healthchecks for both services.
