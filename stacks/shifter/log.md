# Shifter Stack

## Source
https://github.com/TobySuch/Shifter

## Description
Simple self-hosted file-sharing web app built with Django and Tailwind. Upload files, share download links, auto-delete on expiry, multi-user support.

## Services
- **shifter** - Main application (ghcr.io/tobysuch/shifter:latest) on port 8000

## Reference
Compose based on upstream `docker/docker-compose.yml`. Uses SQLite by default; PostgreSQL optional.
