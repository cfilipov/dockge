# Solidus Stack

## Research
- GitHub: solidusio/solidus - Ruby on Rails e-commerce framework
- Has docker-compose.yml in repo with MySQL + PostgreSQL + app (build-based)
- No official pre-built Docker image
- Replaced build-based app with `ruby:3.3-slim`, simplified to PostgreSQL only

## Compose
- Used `ruby:3.3-slim` (no official Solidus image)
- PostgreSQL 16 for database (Solidus supports both MySQL and PostgreSQL)
- Redis 7 for caching/background jobs
- Standard Rails environment variables
