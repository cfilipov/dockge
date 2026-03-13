# Sylius Stack

## Research
- GitHub: Sylius/Sylius - PHP/Symfony-based e-commerce framework
- Found docker-compose.yml in 2.0 branch: app (build), MySQL 5.7, mailhog, blackfire
- No official pre-built Docker image
- Simplified: removed blackfire, upgraded MySQL to 8.0

## Compose
- Based on upstream docker-compose.yml structure
- Used `php:8.3-fpm` (no official Sylius image)
- MySQL 8.0 for database
- MailHog for email testing
