# solidtime

## Source
- GitHub: https://github.com/solidtime-io/solidtime
- Self-hosting examples: https://github.com/solidtime-io/self-hosting-examples
- Docker Hub: solidtime/solidtime

## Research
- Docker Compose from official self-hosting-examples repo (1-docker-with-database)
- laravel.env based on laravel.env.example from same repo
- Five services: app (HTTP), scheduler, queue worker, PostgreSQL 15, Gotenberg
- Image: `solidtime/solidtime:latest`
- Simplified bind-mount volumes to named volumes for portability
- Web UI on port 8000
- APP_KEY must be generated before first run
