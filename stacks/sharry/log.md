# Sharry Stack

## Source
https://github.com/eikek/sharry

## Description
Self-hosted file sharing web application. Authenticated users upload files with optional password and TTL, get a shareable URL. Supports alias pages for receiving files, tus protocol for resumable uploads.

## Services
- **sharry** - Main application (eikek/sharry:latest) on port 9090
- **sharry-db** - PostgreSQL 16 database

## Config Files
- `sharry.conf` - Sharry configuration (HOCON format) with JDBC connection settings

## Reference
Configuration via HOCON file mounted into container. Scala/JVM backend with Elm frontend.
