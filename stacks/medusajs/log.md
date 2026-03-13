# MedusaJs Stack

## Research
- No docker-compose.yml found in the medusajs/medusa repo
- Docker Hub has `medusajs/medusa` image
- Medusa requires PostgreSQL and Redis
- Standard Node.js commerce backend on port 9000

## Changes from upstream
- Created compose.yaml based on Medusa's documented requirements
- Services: medusa (Node.js API), PostgreSQL 15, Redis 7
- All configuration via environment variables with defaults in .env
