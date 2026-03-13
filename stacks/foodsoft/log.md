# Foodsoft

## Sources
- Repository: https://github.com/foodcoops/foodsoft
- Dev compose: https://github.com/foodcoops/foodsoft/blob/master/docker-compose-dev.yml
- Production docs: https://github.com/foodcoops/foodsoft/blob/master/doc/SETUP_PRODUCTION.md
- Docker image: foodcoops/foodsoft (Docker Hub)

## Notes
- Ruby on Rails application for food cooperative management
- Production compose derived from dev compose + production setup docs
- Uses MariaDB 10.5 and Redis 6.2 as dependencies
- Web and worker processes run from the same image with different commands
- SECRET_KEY_BASE must be changed for production use
- Removed dev-only services (phpmyadmin, mailcatcher) and volume mounts
- Removed `version: '2'` field for Compose V2 compatibility
