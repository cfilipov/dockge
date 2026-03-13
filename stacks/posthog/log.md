# PostHog
**Project:** https://github.com/posthog/posthog
**Source:** https://github.com/posthog/posthog
**Status:** done
**Compose source:** Simplified from docker-compose.hobby.yml in repository

## What was done
- Created compose.yaml based on the hobby deployment compose files
- Simplified from 20+ services to core services: PostgreSQL, Redis, ClickHouse, Kafka (Redpanda), MinIO, web, worker, plugins
- Uses posthog/posthog:latest image for app services
- Created .env with database password and secret key

## Issues
- Full hobby deployment has many more services (Temporal, SeaweedFS, etc.); this is a simplified version
