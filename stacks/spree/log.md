# Spree Commerce Stack

## Research
- GitHub: spree/spree - Ruby on Rails e-commerce framework
- Found docker-compose.yml in repo: PostgreSQL 17, Redis 7, Mailpit, web + worker
- Uses build-based app service; replaced with ruby:3.3-slim
- Based closely on the upstream docker-compose.yml structure

## Compose
- Modeled after upstream docker-compose.yml
- Web + Worker + PostgreSQL + Redis + Mailpit
- Used `ruby:3.3-slim` (no official pre-built image)
- Healthchecks on postgres and redis matching upstream
