# MediaCMS Stack

## Source
- GitHub: https://github.com/mediacms-io/mediacms
- Category: Media Streaming - Video Streaming

## Description
MediaCMS is a modern, fully featured open-source video and media CMS. It supports video uploads, transcoding, and streaming with a responsive web interface.

## Stack Components
- **web**: Main MediaCMS web application (Django + Nginx + uWSGI)
- **migrations**: Database migration runner (runs once on startup)
- **celery-beat**: Celery beat scheduler for periodic tasks
- **celery-worker**: Celery worker for async video processing/transcoding
- **db**: PostgreSQL 17 database
- **redis**: Redis for Celery task queue

## Notes
- Based on official docker-compose.yaml from the repository
- Uses shared named volume for MediaCMS data across services
- Migration service runs first and seeds admin user
- Multiple Celery services handle video transcoding pipeline
