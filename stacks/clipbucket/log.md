# ClipBucket

## Overview
ClipBucket is a self-hosted video sharing platform similar to YouTube. Uses PHP/MySQL stack.

## Image
- `xuping/clipbucket:latest` from Docker Hub (community image)
- `mysql:5.7` for database

## Ports
- 8080 → 80 (web UI)

## Volumes
- `clipbucket_data` — uploaded video files
- `clipbucket_uploads` — user uploads
- `db_data` — MySQL database

## Source
- Search (no official Docker image; community images available)
