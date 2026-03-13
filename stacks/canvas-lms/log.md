# Canvas LMS

- **Source**: https://github.com/instructure/canvas-lms
- **Image**: instructure/canvas-lms:stable (Docker Hub)
- **Description**: Open-source LMS developed by Instructure. Rails app with PostgreSQL and Redis.
- **Services**: web (main app), jobs (delayed job worker), postgres, redis
- **Compose reference**: Based on upstream docker-compose.yml (uses build: context, adapted to use published image)
